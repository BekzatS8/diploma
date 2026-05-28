package router

import (
	"log/slog"
	"time"

	"buhpro/internal/common/auth"
	"buhpro/internal/config"
	"buhpro/internal/http/handlers/system"
	"buhpro/internal/http/middleware"
	"buhpro/internal/http/swagger"
	attachmentsmodule "buhpro/internal/modules/attachments"
	authmodule "buhpro/internal/modules/auth"
	chatsmodule "buhpro/internal/modules/chats"
	coursesmodule "buhpro/internal/modules/courses"
	devpaymentsmodule "buhpro/internal/modules/devpayments"
	leadsmodule "buhpro/internal/modules/leads"
	notificationsmodule "buhpro/internal/modules/notifications"
	ordersmodule "buhpro/internal/modules/orders"
	paymentmodule "buhpro/internal/modules/payment"
	profilemodule "buhpro/internal/modules/profile"
	ratingsmodule "buhpro/internal/modules/ratingsanctions"
	responsesmodule "buhpro/internal/modules/responses"
	reviewsmodule "buhpro/internal/modules/reviews"
	selectionmodule "buhpro/internal/modules/selection"
	uploadsmodule "buhpro/internal/modules/uploads"
	walletsmodule "buhpro/internal/modules/wallets"
	"buhpro/internal/platform/metrics"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Deps struct {
	Config               config.Config
	Logger               *slog.Logger
	SystemHandlers       *system.Handler
	JWTManager           *auth.JWTManager
	AuthHandler          *authmodule.Handler
	ProfileHandler       *profilemodule.Handler
	OrdersHandler        *ordersmodule.Handler
	ResponsesHandler     *responsesmodule.Handler
	PaymentHandler       *paymentmodule.Handler
	DevPaymentsHandler   *devpaymentsmodule.Handler
	SelectionHandler     *selectionmodule.Handler
	ReviewsHandler       *reviewsmodule.Handler
	RatingHandler        *ratingsmodule.Handler
	CoursesHandler       *coursesmodule.Handler
	ChatsHandler         *chatsmodule.Handler
	NotificationsHandler *notificationsmodule.Handler
	UploadsHandler       *uploadsmodule.Handler
	AttachmentsHandler   *attachmentsmodule.Handler
	LeadsHandler         *leadsmodule.Handler
	WalletsHandler       *walletsmodule.Handler
	Metrics              *metrics.Metrics
}

func New(deps Deps) *gin.Engine {
	gin.SetMode(mode(deps.Config.App.Env))
	r := gin.New()

	r.Use(middleware.RequestID())
	r.Use(middleware.DebugErrorMiddleware(deps.Config.App.Env != "production"))
	r.Use(middleware.RequestLogging(deps.Logger))
	r.Use(cors.New(cors.Config{
		AllowOrigins:     deps.Config.Server.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.Use(middleware.ValidateUUIDPathParams("id", "userId", "executorId", "responseId", "transactionId", "materialId", "messageId", "reviewId"))

	if deps.Config.Metrics.Enabled && deps.Metrics != nil {
		r.Use(deps.Metrics.Middleware())
		metricsHandlers := []gin.HandlerFunc{}
		if !deps.Config.Metrics.Public {
			metricsHandlers = append(metricsHandlers, middleware.RequireInternalRequest())
		}
		metricsHandlers = append(metricsHandlers, metrics.Handler())
		r.GET(deps.Config.Metrics.Path, metricsHandlers...)
	}

	r.GET("/healthz", deps.SystemHandlers.Healthz)
	r.GET("/readyz", deps.SystemHandlers.Readyz)
	r.Static("/uploads", deps.Config.Storage.LocalPath)
	swagger.Register(r)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", deps.SystemHandlers.Ping)

		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", deps.AuthHandler.Register)
			authGroup.POST("/login", deps.AuthHandler.Login)
			authGroup.POST("/refresh", deps.AuthHandler.Refresh)
			authGroup.Use(middleware.RequireAuth(deps.JWTManager))
			authGroup.POST("/logout", deps.AuthHandler.Logout)
			authGroup.GET("/me", deps.AuthHandler.Me)
		}

		profileGroup := v1.Group("/profile")
		profileGroup.Use(middleware.RequireAuth(deps.JWTManager))
		profileGroup.GET("", deps.ProfileHandler.Get)
		profileGroup.PATCH("", deps.ProfileHandler.Patch)
		profileGroup.PATCH("/avatar", deps.ProfileHandler.SetAvatar)
		profileGroup.DELETE("/avatar", deps.ProfileHandler.ClearAvatar)

		filesGroup := v1.Group("/files")
		filesGroup.Use(middleware.RequireAuth(deps.JWTManager))
		filesGroup.GET("/:id", deps.UploadsHandler.GetByID)
		filesGroup.POST("", deps.UploadsHandler.Upload)
		filesGroup.DELETE("/:id", deps.UploadsHandler.Delete)

		myFilesGroup := v1.Group("/my/files")
		myFilesGroup.Use(middleware.RequireAuth(deps.JWTManager))
		myFilesGroup.GET("", deps.UploadsHandler.ListMy)

		myWalletGroup := v1.Group("/my/wallet")
		myWalletGroup.Use(middleware.RequireAuth(deps.JWTManager))
		myWalletGroup.GET("", deps.WalletsHandler.MyWallet)

		adminWalletsGroup := v1.Group("/admin/wallets")
		adminWalletsGroup.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("admin"))
		adminWalletsGroup.GET("/:userId", deps.WalletsHandler.AdminGet)
		adminWalletsGroup.POST("/:userId/credit", deps.WalletsHandler.AdminCredit)

		attachmentsGroup := v1.Group("/attachments")
		attachmentsGroup.Use(middleware.RequireAuth(deps.JWTManager))
		attachmentsGroup.GET("", deps.AttachmentsHandler.List)
		attachmentsGroup.POST("", deps.AttachmentsHandler.Attach)
		attachmentsGroup.PATCH("/reorder", deps.AttachmentsHandler.Reorder)
		attachmentsGroup.DELETE("/:id", deps.AttachmentsHandler.Delete)

		v1.GET("/reviews", deps.ReviewsHandler.ListByTarget)
		v1.GET("/ratings", deps.ReviewsHandler.GetRatingSummary)
		reviewsGroup := v1.Group("/reviews")
		reviewsGroup.Use(middleware.RequireAuth(deps.JWTManager))
		reviewsGroup.POST("", deps.ReviewsHandler.CreateEntity)

		myReviewsGroup := v1.Group("/my/reviews")
		myReviewsGroup.Use(middleware.RequireAuth(deps.JWTManager))
		myReviewsGroup.GET("", deps.ReviewsHandler.ListAuthored)
		myReviewsGroup.GET("/:reviewId", deps.ReviewsHandler.GetAuthored)
		myReviewsGroup.PATCH("/:reviewId", deps.ReviewsHandler.PatchAuthored)
		myReviewsGroup.DELETE("/:reviewId", deps.ReviewsHandler.DeleteAuthored)

		leadsGroup := v1.Group("/leads")
		leadsGroup.POST("/executor", deps.LeadsHandler.SubmitExecutor)

		adminLeadsGroup := v1.Group("/admin/executor-leads")
		adminLeadsGroup.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("admin"))
		adminLeadsGroup.GET("", deps.LeadsHandler.List)
		adminLeadsGroup.GET("/:id", deps.LeadsHandler.GetByID)
		adminLeadsGroup.PATCH("/:id/status", deps.LeadsHandler.UpdateStatus)
		adminLeadsGroup.POST("/:id/approve", deps.LeadsHandler.Approve)
		adminLeadsGroup.POST("/:id/reject", deps.LeadsHandler.Reject)

		ordersGroup := v1.Group("/orders")
		ordersGroup.Use(middleware.OptionalAuth(deps.JWTManager))
		ordersGroup.GET("", deps.OrdersHandler.ListPublic)
		ordersGroup.GET("/:id", deps.OrdersHandler.GetByID)

		myOrdersGroup := ordersGroup.Group("/my")
		myOrdersGroup.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("client"))
		myOrdersGroup.GET("", deps.OrdersHandler.ListMy)
		myOrdersGroup.GET("/:id", deps.OrdersHandler.GetMyByID)
		myOrdersGroup.PATCH("/:id", deps.OrdersHandler.UpdateMyByID)
		myOrdersGroup.DELETE("/:id", deps.OrdersHandler.DeleteMyByID)
		myOrdersGroup.POST("/:id/submit", deps.OrdersHandler.Submit)
		myOrdersGroup.POST("/:id/cancel", deps.OrdersHandler.Cancel)

		clientOrdersGroup := ordersGroup.Group("")
		clientOrdersGroup.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("client"))
		clientOrdersGroup.POST("", deps.OrdersHandler.Create)
		orderResponsesGroup := ordersGroup.Group("/:id/responses")
		orderResponsesGroup.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("executor"))
		orderResponsesGroup.POST("", deps.ResponsesHandler.Create)
		orderResponsesGroup.GET("/my", deps.ResponsesHandler.ListOrderMy)
		orderResponsesGroup.GET("/my/:responseId", deps.ResponsesHandler.GetOrderMyByID)
		orderResponsesGroup.PATCH("/my/:responseId", deps.ResponsesHandler.UpdateOrderMyByID)
		orderResponsesGroup.DELETE("/my/:responseId", deps.ResponsesHandler.DeleteOrderMyByID)
		orderResponsesGroup.POST("/my/:responseId/submit", deps.ResponsesHandler.Submit)
		orderResponsesGroup.POST("/my/:responseId/cancel", deps.ResponsesHandler.Cancel)

		myResponsesGroup := v1.Group("/my/responses")
		myResponsesGroup.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("executor", "admin"))
		myResponsesGroup.GET("", deps.ResponsesHandler.ListMy)
		myResponsesGroup.GET("/:id", deps.ResponsesHandler.GetMyByID)

		clientResponsesGroup := v1.Group("/client/orders/:id/responses")
		clientResponsesGroup.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("client", "admin"))
		clientResponsesGroup.GET("", deps.ResponsesHandler.ListClientOrder)
		clientResponsesGroup.GET("/:responseId", deps.ResponsesHandler.GetClientOrderByID)

		paymentGroup := v1.Group("/payment")
		paymentGroup.POST("/webhook", deps.PaymentHandler.Webhook)
		paymentGroup.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("client"))
		paymentGroup.POST("/create", deps.PaymentHandler.Create)

		devPaymentsGroup := v1.Group("/dev/payments")
		devPaymentsGroup.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("admin"))
		devPaymentsGroup.POST("/:transactionId/confirm", deps.DevPaymentsHandler.Confirm)
		devPaymentsGroup.POST("/:transactionId/fail", deps.DevPaymentsHandler.Fail)
		clientOrderLifecycle := v1.Group("/client/orders/:id")
		clientOrderLifecycle.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("client", "admin"))
		clientOrderLifecycle.POST("/select-response/:responseId", deps.SelectionHandler.SelectResponse)
		clientOrderLifecycle.GET("/selection", deps.SelectionHandler.GetSelection)
		clientOrderLifecycle.POST("/complete", deps.SelectionHandler.Complete)
		clientOrderLifecycle.POST("/reopen", deps.SelectionHandler.Reopen)
		clientOrderLifecycle.POST("/review", deps.ReviewsHandler.Create)
		clientOrderLifecycle.GET("/review", deps.ReviewsHandler.GetByOrder)

		orderReviewLifecycle := v1.Group("/orders/:id/review")
		orderReviewLifecycle.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("client", "executor"))
		orderReviewLifecycle.POST("", deps.ReviewsHandler.Create)

		v1.GET("/executors/:executorId/reviews", deps.ReviewsHandler.ListExecutor)
		v1.GET("/users/:userId/reviews", deps.ReviewsHandler.ListUser)
		v1.GET("/executors/:executorId/rating", deps.RatingHandler.GetRating)

		mySanctions := v1.Group("/my/sanctions")
		mySanctions.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("executor", "admin"))
		mySanctions.GET("", deps.RatingHandler.MySanctions)

		adminSanctions := v1.Group("/admin/sanctions")
		adminSanctions.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("admin"))
		adminSanctions.GET("", deps.RatingHandler.AdminSanctions)
		adminSanctions.POST("/expire", deps.RatingHandler.ExpireDue)
		adminSanctions.GET("/:id", deps.RatingHandler.AdminSanctionByID)
		adminSanctions.POST("/:id/resolve", deps.RatingHandler.Resolve)
		adminSanctions.POST("/:id/lift", deps.RatingHandler.Lift)

		coachCourses := v1.Group("/coach/courses")
		coachCourses.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("coach", "executor", "admin"))
		coachCourses.POST("", deps.CoursesHandler.CreateCourse)
		coachCourses.PATCH("/:id", deps.CoursesHandler.PatchCourse)
		coachCourses.GET("/analytics", deps.CoursesHandler.CreatorAnalytics)
		coachCourses.GET("/:id/students", deps.CoursesHandler.ListCourseStudents)
		coachCourses.GET("/:id", deps.CoursesHandler.GetCoachCourse)
		coachCourses.GET("", deps.CoursesHandler.ListCoachCourses)
		coachCourses.POST("/:id/publish", deps.CoursesHandler.PublishCourse)
		coachCourses.POST("/:id/archive", deps.CoursesHandler.ArchiveCourse)
		coachCourses.DELETE("/:id", deps.CoursesHandler.DeleteCourse)
		coachCourses.POST("/:id/materials", deps.CoursesHandler.CreateMaterial)
		coachCourses.PATCH("/:id/materials/:materialId", deps.CoursesHandler.PatchMaterial)
		coachCourses.DELETE("/:id/materials/:materialId", deps.CoursesHandler.DeleteMaterial)

		coursesCatalog := v1.Group("/courses")
		coursesCatalog.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("executor", "coach", "admin"))
		coursesCatalog.GET("", deps.CoursesHandler.ListCourses)
		coursesCatalog.GET("/:id", deps.CoursesHandler.GetCourse)

		adminCourseAssignments := v1.Group("/admin/course-assignments")
		adminCourseAssignments.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("admin"))
		adminCourseAssignments.POST("", deps.CoursesHandler.CreateAssignment)
		adminCourseAssignments.GET("", deps.CoursesHandler.ListAdminAssignments)

		adminCourses := v1.Group("/admin/courses")
		adminCourses.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("admin"))
		adminCourses.GET("", deps.CoursesHandler.ListAdminCourses)
		adminCourses.POST("/:id/approve", deps.CoursesHandler.ApproveCourseModeration)
		adminCourses.POST("/:id/reject", deps.CoursesHandler.RejectCourseModeration)

		myCourseAssignments := v1.Group("/my/course-assignments")
		myCourseAssignments.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("executor"))
		myCourseAssignments.GET("", deps.CoursesHandler.ListMyAssignments)
		myCourseAssignments.POST("/enroll", deps.CoursesHandler.EnrollMyCourse)
		myCourseAssignments.GET("/:id", deps.CoursesHandler.GetMyAssignment)
		myCourseAssignments.POST("/:id/materials/:materialId/complete", deps.CoursesHandler.MarkMaterialCompleted)
		myCourseAssignments.POST("/:id/mark-completed", deps.CoursesHandler.MarkCompleted)

		myNotifications := v1.Group("/my/notifications")
		myNotifications.Use(middleware.RequireAuth(deps.JWTManager))
		myNotifications.GET("", deps.NotificationsHandler.ListMy)
		myNotifications.GET("/:id", deps.NotificationsHandler.GetMyByID)
		myNotifications.POST("/:id/read", deps.NotificationsHandler.MarkRead)
		myNotifications.POST("/read-all", deps.NotificationsHandler.MarkAllRead)

		adminNotifications := v1.Group("/admin/notifications")
		adminNotifications.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("admin"))
		adminNotifications.GET("", deps.NotificationsHandler.ListAdmin)
		adminNotifications.GET("/:id", deps.NotificationsHandler.GetAdminByID)

		myChats := v1.Group("/my/chats")
		myChats.Use(middleware.RequireAuth(deps.JWTManager))
		myChats.GET("", deps.ChatsHandler.ListMyChats)
		myChats.POST("", deps.ChatsHandler.CreateDirectChat)
		myChats.GET("/:id", deps.ChatsHandler.GetMyChatByID)
		myChats.GET("/:id/messages", deps.ChatsHandler.ListMyMessages)
		myChats.POST("/:id/messages", deps.ChatsHandler.SendMyMessage)
		myChats.PATCH("/:id/messages/:messageId", deps.ChatsHandler.PatchMyMessage)
		myChats.DELETE("/:id/messages/:messageId", deps.ChatsHandler.DeleteMyMessage)
		myChats.POST("/:id/read", deps.ChatsHandler.MarkMyChatRead)

		adminChats := v1.Group("/admin/chats")
		adminChats.Use(middleware.RequireAuth(deps.JWTManager), middleware.RequireRoles("admin"))
		adminChats.GET("", deps.ChatsHandler.ListAdminChats)
		adminChats.GET("/:id", deps.ChatsHandler.GetAdminChatByID)
		adminChats.GET("/:id/messages", deps.ChatsHandler.ListAdminMessages)
	}

	return r
}

func mode(env string) string {
	if env == "production" {
		return gin.ReleaseMode
	}
	return gin.DebugMode
}
