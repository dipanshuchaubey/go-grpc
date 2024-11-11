package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	ot "user-service/otel"
	as "user-service/protos/auth_service"
	bs "user-service/protos/bootcamp_service"
	us "user-service/protos/user_service"
	types "user-service/types"

	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	name   = "user-service"
	tracer = otel.Tracer(name)
	meter  = otel.Meter(name)
	logger = otelslog.NewLogger(name)
)

type server struct {
	us.UnimplementedUserServiceServer
	as.UnimplementedAuthServiceServer
	bs.UnimplementedBootcampServiceServer
}

func (s *server) GetUser(ctx context.Context, in *us.GetUserRequest) (*us.GetUserResponse, error) {
	// fmt.Println(fmt.Sprintf("GetUser: params - TenantID: %s, UserID: %s", in.TenantId, in.UserId))
	deadline, _ := ctx.Deadline()
	fmt.Println("Time remaining: ", time.Until(deadline))

	return &us.GetUserResponse{
		Success: true,
		Data: &us.UserInfo{
			UserId:   12,
			FullName: "Dipanshu",
			Email:    "dipanshu@gmail.com",
		},
	}, nil
}

func (s *server) GetUsers(ctx context.Context, in *us.GetUsersRequest) (*us.GetUsersResponse, error) {
	_, span := tracer.Start(ctx, "GetUsers")
	defer span.End()

	logger.InfoContext(ctx, "GetUsers: params - TenantID")

	httpRes, httpErr := http.Get("https://jsonplaceholder.typicode.com/users")

	if httpErr != nil {
		log.Fatalf("Cannot make http request %v", httpErr)
	}

	resBody, resErr := io.ReadAll(httpRes.Body)
	if resErr != nil {
		log.Fatal("Cannot read response body")
	}

	var userInfos []*types.UserInfo
	marErr := json.Unmarshal(resBody, &userInfos)

	if marErr != nil {
		log.Fatalf("Failed to unmarshal json response %v", marErr)
	}

	var response []*us.UserInfo
	for _, user := range userInfos {
		response = append(response, &us.UserInfo{
			UserId:   user.ID,
			Email:    user.Email,
			FullName: user.Name,
			UserType: us.UserTypes_USER_TYPE_ACTIVE,
		})
	}

	return &us.GetUsersResponse{
		Success: true,
		Data:    response,
	}, nil
}

func (s *server) Login(ctx context.Context, in *as.LoginRequest) (*as.LoginResponse, error) {
	// fmt.Println(fmt.Sprintf("Login: params - username: %s, email: %s", in.Username, in.Email))

	return &as.LoginResponse{
		Success: true,
		Token:   "Sample token",
		Expiry:  timestamppb.New(time.Now().AddDate(0, 0, 1)),
	}, nil
}

func (s *server) GetBootcampsDetails(ctx context.Context, in *bs.GetBootcampsDetailsRequest) (*bs.GetBootcampsDetailsResponse, error) {
	fmt.Println("GetBootcampsDetails - params BootcampIDs: ", in.BootcampIds)

	bootcampRes, httpErr := http.Get("https://bootcamper.dipanshu.work/api/v1/bootcamps")
	if httpErr != nil {
		errMsg := fmt.Errorf("GetBootcampsDetails: error getting bootcamps %v", httpErr)
		fmt.Println(errMsg)
		return nil, errMsg
	}

	resBody, resErr := io.ReadAll(bootcampRes.Body)
	if resErr != nil {
		log.Fatal("Cannot read response body")
	}

	var bootcampInfos types.BootcampResponse
	marErr := json.Unmarshal(resBody, &bootcampInfos)

	if marErr != nil {
		errMsg := fmt.Errorf("GetBootcampsDetails: error unmarshalling json data %v", marErr)
		fmt.Println(errMsg)
		return nil, errMsg
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	// var response bs.GetBootcampsDetailsResponse

	for _, bootcampInfo := range bootcampInfos.Data {
		wg.Add(1)

		fmt.Println("GetBootcampsDetails: fetching reviews for bootcampID: ", bootcampInfo.ID)

		go func() {
			defer wg.Done()

			defer func() {
				if err := recover(); err != nil {
					fmt.Println("recovered from panic!!")
				}
			}()

			reviewsRes, httpErr := http.Get(fmt.Sprintf("https://bootcamper.dipanshu.work/api/v1/bootcamps/%s/reviews", bootcampInfo.ID))
			if httpErr != nil {
				errMsg := fmt.Errorf("GetBootcampsDetails: GetBootcampsDetails: error getting reviews for BootcampID %v: %v", bootcampInfo.ID, marErr)
				fmt.Println(errMsg)
			}

			resBody, resErr := io.ReadAll(reviewsRes.Body)
			if resErr != nil {
				log.Fatal("Cannot read response body")
			}

			var reviews types.ReviewResponse
			marErr := json.Unmarshal(resBody, &reviews)
			if marErr != nil {
				errMsg := fmt.Errorf("GetBootcampsDetails: error unmarshalling json data %v", marErr)
				fmt.Println(errMsg)
			}

			mu.Lock()
			defer mu.Unlock()

			for i, review := range reviews.Data {
				fmt.Println("Review ", i, ":", review.Title)
			}
		}()
	}

	wg.Wait()

	return &bs.GetBootcampsDetailsResponse{}, nil
}

func timeoutInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*2)
	defer cancel()

	// Create a channel to catch the result or a panic recovery
	done := make(chan struct{})
	var resp interface{}
	var err error

	// Set up OpenTelemetry.
	otelShutdown, otelErr := ot.SetupOTelSDK(ctx)
	if otelErr != nil {
		return nil, otelErr
	}

	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in %s: %v", info.FullMethod, r)
				err = status.Error(codes.Internal, "internal server error")
			}
			close(done)
		}()
		resp, err = handler(ctx, req)
		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		// Handle context timeout
		log.Printf("Timeout in %s", info.FullMethod)
		return nil, status.Error(codes.DeadlineExceeded, "request timed out")
	case <-done:
		// Proceed normally if no timeout
		return resp, err
	}
}

func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	// Channel to listen for termination signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		defer wg.Done()
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("recovered from panic:", err)
			}
		}()

		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("failed to listen on port 50051: %v", err)
		}

		options := []grpc.ServerOption{
			grpc.ConnectionTimeout(time.Second),
			grpc.UnaryInterceptor(timeoutInterceptor),
		}

		s := grpc.NewServer(options...)

		us.RegisterUserServiceServer(s, &server{})
		as.RegisterAuthServiceServer(s, &server{})
		bs.RegisterBootcampServiceServer(s, &server{})

		reflection.Register(s)

		go func() {
			log.Printf("gRPC server listening at %v", lis.Addr())
			if err := s.Serve(lis); err != nil {
				log.Fatalf("failed to serve: %v", err)
			}
		}()

		<-stop // Wait for stop signal
		log.Println("Shutting down gRPC server...")
		s.GracefulStop()
	}()

	wg.Wait()
}
