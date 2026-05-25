package swagger

import (
	"net/http"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
)

const uiHTML = `<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>BuhPro API Swagger</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: "/swagger/doc.json",
      dom_id: "#swagger-ui",
      deepLinking: true,
      persistAuthorization: true
    });
  </script>
</body>
</html>`

// Register exposes the generated OpenAPI document and a Swagger UI page.
func Register(r *gin.Engine) {
	r.GET("/swagger/doc.json", func(c *gin.Context) {
		c.JSON(http.StatusOK, Spec())
	})
	r.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	r.GET("/swagger/index.html", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(uiHTML))
	})
}

// Spec returns a complete OpenAPI 3.0 description for the current REST API.
func Spec() map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "BuhPro Backend API",
			"version":     "1.0.0",
			"description": "REST API for BuhPro MVP: auth, profiles, orders, paid responses, selection, reviews, sanctions, courses, notifications and chats.",
		},
		"servers": []any{
			map[string]any{"url": "http://localhost:8080", "description": "Local Docker/dev server"},
		},
		"tags": []any{
			tag("System", "Health, readiness, metrics and ping endpoints."),
			tag("Auth", "Registration, login, token refresh, logout and current user."),
			tag("Profile", "Current role profile read/update."),
			tag("Uploads", "Local file upload metadata and private owner file management."),
			tag("Attachments", "Polymorphic one-to-many file links for domain entities."),
			tag("Wallets", "Internal demo currency balances and admin top-ups."),
			tag("Executor Leads", "Public executor registration with document verification."),
			tag("Admin Executor Leads", "Admin review and conversion of executor leads."),
			tag("Orders", "Client order creation, management and public feed."),
			tag("Responses", "Executor paid responses and client response review."),
			tag("Payments", "YooKassa payment creation and webhook processing."),
			tag("Dev Payments", "Development-only payment confirmation/failure endpoints."),
			tag("Selection & Lifecycle", "Client selection, completion and reopen flow."),
			tag("Reviews", "Order reviews plus generic target reviews and ratings."),
			tag("Rating & Sanctions", "Executor rating and sanction administration."),
			tag("Courses", "Coach/admin course management and executor assignments."),
			tag("Notifications", "In-app notification read models."),
			tag("Chats", "REST chat access for participants and admins."),
		},
		"paths":      paths(),
		"components": components(),
	}
}

func paths() map[string]any {
	return map[string]any{
		"/healthz":     get("System", "Health check", "Returns liveness status without requiring database access.", false, nil, nil, ok("HealthResponse")),
		"/readyz":      get("System", "Readiness check", "Checks database connectivity and reports whether the API is ready to serve traffic.", false, nil, nil, ok("HealthResponse")),
		"/metrics":     get("System", "Prometheus metrics", "Prometheus scrape endpoint when METRICS_ENABLED=true. By default it is available only from internal/private client IPs; set METRICS_PUBLIC=true only for demo exposure.", false, nil, nil, textOK()),
		"/api/v1/ping": get("System", "API ping", "Versioned API ping endpoint.", false, nil, nil, ok("PingResponse")),

		"/api/v1/auth/register": post("Auth", "Register user", "Creates a client or coach user and its role profile in one transaction. Executors must use the executor lead endpoint with documents.", false, nil, body("RegisterRequest"), created("AuthResponse")),
		"/api/v1/auth/login":    post("Auth", "Login", "Authenticates by email and password and returns access/refresh tokens.", false, nil, body("LoginRequest"), ok("AuthResponse")),
		"/api/v1/auth/refresh":  post("Auth", "Refresh tokens", "Rotates a valid refresh token and returns a new token pair.", false, nil, body("RefreshRequest"), ok("TokenPair")),
		"/api/v1/auth/logout":   post("Auth", "Logout", "Revokes the current refresh token.", true, nil, body("LogoutRequest"), ok("StatusResponse")),
		"/api/v1/auth/me":       get("Auth", "Current user", "Returns the authenticated user.", true, nil, nil, ok("MeResponse")),

		"/api/v1/profile": pathItem(
			getOp("Profile", "Get current profile", "Returns the profile table matching the current user's role.", true, nil, nil, ok("ProfileResponse")),
			patchOp("Profile", "Update current profile", "Partially updates the current role profile and returns the updated profile payload.", true, nil, body("UpdateProfileRequest"), ok("ProfileResponse")),
		),
		"/api/v1/profile/avatar": pathItem(
			patchOp("Profile", "Set profile avatar", "Links one uploaded file as the current role profile avatar and returns the updated profile payload.", true, nil, body("SetAvatarRequest"), ok("ProfileResponse")),
			deleteOp("Profile", "Clear profile avatar", "Removes the avatar link from the current role profile without deleting the uploaded file and returns the updated profile payload.", true, nil, nil, ok("ProfileResponse")),
		),

		"/api/v1/files":      post("Uploads", "Upload files", "Uploads one or more files to local storage. Send multipart/form-data with repeated file fields.", true, nil, multipartFilesBody(), created("UploadListResponse")),
		"/api/v1/files/{id}": pathItem(getOp("Uploads", "Get uploaded file metadata", "Returns upload metadata and local URL for the owner; admin can read any upload.", true, []any{path("id", "string", "Upload UUID.")}, nil, ok("UploadView")), deleteOp("Uploads", "Delete uploaded file", "Deletes an owned upload record and local file; admin can delete any upload.", true, []any{path("id", "string", "Upload UUID.")}, nil, ok("StatusResponse"))),
		"/api/v1/my/files":   get("Uploads", "My uploaded files", "Lists uploads owned by the authenticated user.", true, nil, nil, ok("UploadListResponse")),

		"/api/v1/my/wallet":                     get("Wallets", "My wallet", "Returns current user's internal balance and latest transactions.", true, nil, nil, ok("WalletResponse")),
		"/api/v1/admin/wallets/{userId}":        get("Wallets", "Admin get wallet", "Admin reads any user's wallet and latest transactions.", true, []any{path("userId", "string", "User UUID.")}, nil, ok("WalletResponse")),
		"/api/v1/admin/wallets/{userId}/credit": post("Wallets", "Admin credit wallet", "Admin adds internal demo currency to a user wallet.", true, []any{path("userId", "string", "User UUID.")}, body("CreditWalletRequest"), ok("WalletCreditResponse")),

		"/api/v1/attachments": pathItem(
			getOp("Attachments", "List attachments", "Lists files attached to a target entity by target_type and target_id according to target visibility rules.", true, []any{query("target_type", "string", true, "Attachment target type."), query("target_id", "string", true, "Target UUID.")}, nil, ok("AttachmentListResponse")),
			postOp("Attachments", "Attach files", "Links existing uploaded files to one target entity. Owner must own uploads; admin can link any upload.", true, nil, body("AttachRequest"), created("AttachmentListResponse")),
		),
		"/api/v1/attachments/reorder": patchOp("Attachments", "Reorder attachments", "Rewrites sort_order by the provided attachment id order.", true, nil, body("ReorderAttachmentsRequest"), ok("StatusResponse")),
		"/api/v1/attachments/{id}":    pathItem(deleteOp("Attachments", "Delete attachment link", "Deletes only the attachment link; the uploaded file remains in storage.", true, []any{path("id", "string", "Attachment UUID.")}, nil, ok("StatusResponse"))),

		"/api/v1/leads/executor": post("Executor Leads", "Submit executor registration lead", "Public multipart endpoint for executor registration. Requires identity_document and education_document files; ip_registration_document is optional.", false, nil, executorLeadMultipartBody(), created("ExecutorLeadSubmittedResponse")),
		"/api/v1/admin/executor-leads": get("Admin Executor Leads", "List executor leads", "Admin list of executor registration leads.", true, []any{
			query("status", "string", false, "Optional lead status."), pageParam(), pageSizeParam(),
		}, nil, listOK("ExecutorLeadListResponse")),
		"/api/v1/admin/executor-leads/{id}":         get("Admin Executor Leads", "Get executor lead", "Admin detail view with submitted documents.", true, []any{path("id", "string", "Lead UUID.")}, nil, ok("ExecutorLeadView")),
		"/api/v1/admin/executor-leads/{id}/status":  patchOp("Admin Executor Leads", "Update executor lead status", "Marks a lead as new, in_review or approved without conversion.", true, []any{path("id", "string", "Lead UUID.")}, body("UpdateLeadStatusRequest"), ok("StatusResponse")),
		"/api/v1/admin/executor-leads/{id}/approve": post("Admin Executor Leads", "Approve executor lead", "Converts a verified lead into an executor user, executor profile, uploads and profile_document attachments.", true, []any{path("id", "string", "Lead UUID.")}, body("ApproveLeadRequest"), created("ApproveLeadResponse")),
		"/api/v1/admin/executor-leads/{id}/reject":  post("Admin Executor Leads", "Reject executor lead", "Rejects a lead with a reason.", true, []any{path("id", "string", "Lead UUID.")}, body("RejectLeadRequest"), ok("StatusResponse")),

		"/api/v1/orders": pathItem(
			getOp("Orders", "Public orders feed", "Lists published, non-deleted orders with category, budget and text filters.", false, []any{
				query("category", "string", false, "Category slug."),
				query("budget_min", "number", false, "Minimum order budget."),
				query("budget_max", "number", false, "Maximum order budget."),
				query("deadline_before", "string", false, "RFC3339 deadline filter."),
				query("region", "string", false, "Region filter; online orders also match."),
				query("q", "string", false, "Search in title and description."),
				pageParam(), pageSizeParam(),
			}, nil, listOK("OrdersListResponse")),
			postOp("Orders", "Create draft order", "Client-only endpoint that creates an order in draft status.", true, nil, body("CreateOrderRequest"), created("OrderResponse")),
		),
		"/api/v1/orders/{id}": get("Orders", "Get order", "Returns a published order publicly; owner client and admin can also read non-published orders.", false, []any{path("id", "string", "Order UUID.")}, nil, ok("OrderResponse")),
		"/api/v1/orders/my": get("Orders", "My orders", "Lists authenticated client orders.", true, []any{
			query("status", "string", false, "Optional order status."),
			pageParam(), pageSizeParam(),
		}, nil, listOK("OrdersListResponse")),
		"/api/v1/orders/my/{id}": pathItem(
			getOp("Orders", "My order details", "Returns owned order details and latest payment transaction when present.", true, []any{path("id", "string", "Order UUID.")}, nil, ok("OrderDetailsResponse")),
			patchOp("Orders", "Update draft order", "Updates an owned order while it is still in draft status.", true, []any{path("id", "string", "Order UUID.")}, body("UpdateOrderRequest"), ok("OrderResponse")),
			deleteOp("Orders", "Delete draft/cancelled order", "Soft-deletes an owned draft or cancelled order.", true, []any{path("id", "string", "Order UUID.")}, nil, ok("StatusResponse")),
		),
		"/api/v1/orders/my/{id}/submit": post("Orders", "Submit order for posting payment", "Moves a draft order to payment_pending and creates an order_posting payment transaction.", true, []any{path("id", "string", "Order UUID.")}, nil, ok("SubmitOrderResponse")),
		"/api/v1/orders/my/{id}/cancel": post("Orders", "Cancel order", "Cancels an owned order when its current status allows cancellation.", true, []any{path("id", "string", "Order UUID.")}, nil, ok("OrderResponse")),

		"/api/v1/orders/{id}/responses": post("Responses", "Create draft response", "Executor-only endpoint that creates a draft response for a published order.", true, []any{path("id", "string", "Order UUID.")}, body("CreateResponseRequest"), created("ResponseView")),
		"/api/v1/orders/{id}/responses/my": get("Responses", "My responses in order", "Lists the current executor's responses for a specific order.", true, []any{
			path("id", "string", "Order UUID."), query("status", "string", false, "Optional response status."), pageParam(), pageSizeParam(),
		}, nil, listOK("ResponsesListResponse")),
		"/api/v1/orders/{id}/responses/my/{responseId}": pathItem(
			getOp("Responses", "My response details", "Returns the current executor's response for an order.", true, []any{path("id", "string", "Order UUID."), path("responseId", "string", "Response UUID.")}, nil, ok("ResponseView")),
			patchOp("Responses", "Update draft response", "Updates a response while it is in draft status.", true, []any{path("id", "string", "Order UUID."), path("responseId", "string", "Response UUID.")}, body("UpdateResponseRequest"), ok("ResponseView")),
			deleteOp("Responses", "Delete response", "Soft-deletes a draft or cancelled response.", true, []any{path("id", "string", "Order UUID."), path("responseId", "string", "Response UUID.")}, nil, ok("StatusResponse")),
		),
		"/api/v1/orders/{id}/responses/my/{responseId}/submit": post("Responses", "Submit response for payment", "Moves a draft response to payment_pending and creates a response_submission payment transaction.", true, []any{path("id", "string", "Order UUID."), path("responseId", "string", "Response UUID.")}, nil, ok("SubmitResponsePayload")),
		"/api/v1/orders/{id}/responses/my/{responseId}/cancel": post("Responses", "Cancel response", "Cancels an executor response from draft or payment_pending.", true, []any{path("id", "string", "Order UUID."), path("responseId", "string", "Response UUID.")}, nil, ok("ResponseView")),
		"/api/v1/my/responses": get("Responses", "My responses", "Executor list of own responses; admin receives all responses.", true, []any{
			query("status", "string", false, "Optional response status."), pageParam(), pageSizeParam(),
		}, nil, listOK("ResponsesListResponse")),
		"/api/v1/my/responses/{id}":                         get("Responses", "My response by id", "Executor reads own response; admin reads any response.", true, []any{path("id", "string", "Response UUID.")}, nil, ok("ResponseView")),
		"/api/v1/client/orders/{id}/responses":              get("Responses", "Client view order responses", "Client/admin list of paid submitted responses for an owned order.", true, []any{path("id", "string", "Order UUID."), pageParam(), pageSizeParam()}, nil, listOK("ResponsesListResponse")),
		"/api/v1/client/orders/{id}/responses/{responseId}": get("Responses", "Client response details", "Client/admin details for a paid submitted response on an owned order.", true, []any{path("id", "string", "Order UUID."), path("responseId", "string", "Response UUID.")}, nil, ok("ResponseView")),

		"/api/v1/payment/create":  post("Payments", "Create YooKassa payment", "Client endpoint that validates the requested amount against the server-calculated order charge, creates a YooKassa payment and returns confirmation_url for frontend redirect.", true, nil, body("CreatePaymentRequest"), ok("CreatePaymentResponse")),
		"/api/v1/payment/webhook": post("Payments", "YooKassa webhook", "Public webhook endpoint for YooKassa notifications. Handles payment.succeeded and confirms the related pending transaction.", false, nil, body("YooKassaWebhook"), ok("StatusResponse")),

		"/api/v1/dev/payments/{transactionId}/confirm": post("Dev Payments", "Confirm dev payment", "Admin-only non-production endpoint that marks a pending payment as succeeded and advances the related order/response.", true, []any{path("transactionId", "string", "Payment transaction UUID.")}, nil, ok("StatusResponse")),
		"/api/v1/dev/payments/{transactionId}/fail":    post("Dev Payments", "Fail dev payment", "Admin-only non-production endpoint that marks a pending payment as failed and rolls the related order/response back to draft.", true, []any{path("transactionId", "string", "Payment transaction UUID.")}, nil, ok("StatusResponse")),

		"/api/v1/client/orders/{id}/select-response/{responseId}": post("Selection & Lifecycle", "Select response", "Client/admin selects a paid submitted response, moves the order to in_progress, accepts the chosen response, rejects other submitted responses and creates a chat.", true, []any{path("id", "string", "Order UUID."), path("responseId", "string", "Response UUID.")}, nil, ok("StatusResponse")),
		"/api/v1/client/orders/{id}/selection":                    get("Selection & Lifecycle", "Get selection", "Returns current selected response/executor snapshot for an owned order.", true, []any{path("id", "string", "Order UUID.")}, nil, ok("Selection")),
		"/api/v1/client/orders/{id}/complete":                     post("Selection & Lifecycle", "Complete order", "Moves an in_progress selected order to completed.", true, []any{path("id", "string", "Order UUID.")}, nil, ok("StatusResponse")),
		"/api/v1/client/orders/{id}/reopen":                       post("Selection & Lifecycle", "Reopen order", "MVP/dev simplification: moves a completed order back to in_progress when no review exists.", true, []any{path("id", "string", "Order UUID.")}, nil, ok("StatusResponse")),
		"/api/v1/client/orders/{id}/review": pathItem(
			getOp("Reviews", "Get order review", "Client/admin reads the review for an owned order.", true, []any{path("id", "string", "Order UUID.")}, nil, ok("Review")),
			postOp("Reviews", "Create review", "Client/admin creates one review for a completed order and triggers rating/sanction recalculation.", true, []any{path("id", "string", "Order UUID.")}, body("CreateReviewRequest"), created("Review")),
		),
		"/api/v1/reviews": pathItem(
			getOp("Reviews", "List target reviews", "Lists generic reviews attached to any supported target_type and target_id.", false, []any{query("target_type", "string", true, "Review target type."), query("target_id", "string", true, "Target UUID."), pageParam(), pageSizeParam()}, nil, listOK("EntityReviewsListResponse")),
			postOp("Reviews", "Create target review", "Creates a generic course review after the current executor has completed an assignment for that course.", true, nil, body("CreateEntityReviewRequest"), created("EntityReview")),
		),
		"/api/v1/ratings":                        get("Reviews", "Target rating summary", "Returns aggregate rating for any supported target_type and target_id.", false, []any{query("target_type", "string", true, "Review target type."), query("target_id", "string", true, "Target UUID.")}, nil, ok("EntityRatingSummary")),
		"/api/v1/executors/{executorId}/reviews": get("Reviews", "Executor reviews", "Public paginated reviews for an executor, newest first.", false, []any{path("executorId", "string", "Executor user UUID."), pageParam(), pageSizeParam()}, nil, listOK("ReviewsListResponse")),
		"/api/v1/executors/{executorId}/rating":  get("Rating & Sanctions", "Executor rating", "Public rating summary for an executor.", false, []any{path("executorId", "string", "Executor user UUID.")}, nil, ok("RatingInfo")),
		"/api/v1/my/sanctions":                   get("Rating & Sanctions", "My sanctions", "Executor/admin list of sanctions for the current user.", true, nil, nil, ok("SanctionsResponse")),
		"/api/v1/admin/sanctions":                get("Rating & Sanctions", "Admin list sanctions", "Admin paginated sanction list.", true, []any{pageParam(), pageSizeParam()}, nil, listOK("AdminSanctionsResponse")),
		"/api/v1/admin/sanctions/expire":         post("Rating & Sanctions", "Expire due sanctions", "Admin recalculates active sanctions and marks due ones as expired.", true, nil, nil, ok("ExpireSanctionsResponse")),
		"/api/v1/admin/sanctions/{id}":           get("Rating & Sanctions", "Admin get sanction", "Admin sanction details.", true, []any{path("id", "string", "Sanction UUID.")}, nil, ok("Sanction")),
		"/api/v1/admin/sanctions/{id}/resolve":   post("Rating & Sanctions", "Admin resolve sanction", "Marks an expired sanction as resolved by admin.", true, []any{path("id", "string", "Sanction UUID.")}, nil, ok("StatusResponse")),
		"/api/v1/admin/sanctions/{id}/lift":      post("Rating & Sanctions", "Admin lift sanction", "Backward-compatible manual lift endpoint for active or expired sanctions.", true, []any{path("id", "string", "Sanction UUID.")}, nil, ok("StatusResponse")),

		"/api/v1/coach/courses": pathItem(
			getOp("Courses", "Creator list courses", "Coach/admin and eligible executors list owned courses.", true, []any{query("status", "string", false, "Optional course status."), query("category", "string", false, "Optional course category."), query("q", "string", false, "Search in title/subtitle/description."), pageParam(), pageSizeParam()}, nil, listOK("CoursesListResponse")),
			postOp("Courses", "Create course", "Coach/admin and eligible executors create a draft course.", true, nil, body("CreateCourseRequest"), created("Course")),
		),
		"/api/v1/coach/courses/analytics": get("Courses", "Creator course analytics", "CRM summary for owned courses, materials, assignments and student progress.", true, nil, nil, ok("CreatorAnalytics")),
		"/api/v1/coach/courses/{id}": pathItem(
			getOp("Courses", "Creator get course", "Creator/admin reads a course and its materials.", true, []any{path("id", "string", "Course UUID.")}, nil, ok("CourseDetailResponse")),
			patchOp("Courses", "Patch course", "Creator/admin updates a course.", true, []any{path("id", "string", "Course UUID.")}, body("UpdateCourseRequest"), ok("Course")),
		),
		"/api/v1/coach/courses/{id}/students":  get("Courses", "Course students", "Lists assigned executors with course progress for the creator CRM.", true, []any{path("id", "string", "Course UUID."), pageParam(), pageSizeParam()}, nil, listOK("CourseStudentsListResponse")),
		"/api/v1/coach/courses/{id}/publish":   post("Courses", "Publish course", "Moves a draft course to published.", true, []any{path("id", "string", "Course UUID.")}, nil, ok("Course")),
		"/api/v1/coach/courses/{id}/archive":   post("Courses", "Archive course", "Moves a published course to archived.", true, []any{path("id", "string", "Course UUID.")}, nil, ok("Course")),
		"/api/v1/coach/courses/{id}/materials": post("Courses", "Create material", "Adds a material to a course. video/pdf/link require url or upload_id; text requires content.", true, []any{path("id", "string", "Course UUID.")}, body("CreateMaterialRequest"), created("CourseMaterial")),
		"/api/v1/coach/courses/{id}/materials/{materialId}": pathItem(
			patchOp("Courses", "Patch material", "Updates a course material.", true, []any{path("id", "string", "Course UUID."), path("materialId", "string", "Material UUID.")}, body("UpdateMaterialRequest"), ok("CourseMaterial")),
			deleteOp("Courses", "Delete material", "Deletes a course material.", true, []any{path("id", "string", "Course UUID."), path("materialId", "string", "Material UUID.")}, nil, ok("StatusResponse")),
		),
		"/api/v1/courses":      get("Courses", "Course catalog", "Authenticated executor/coach/admin course catalog. Executors only see published courses.", true, []any{query("status", "string", false, "Optional course status; executor is forced to published."), query("category", "string", false, "Optional course category."), query("q", "string", false, "Search in title/subtitle/description."), pageParam(), pageSizeParam()}, nil, listOK("CoursesListResponse")),
		"/api/v1/courses/{id}": get("Courses", "Catalog course details", "Authenticated course details and materials.", true, []any{path("id", "string", "Course UUID.")}, nil, ok("CourseDetailResponse")),
		"/api/v1/admin/course-assignments": pathItem(
			getOp("Courses", "Admin list course assignments", "Admin list with executor, course, status and source filters.", true, []any{
				query("executor_id", "string", false, "Executor user UUID."), query("course_id", "string", false, "Course UUID."),
				query("status", "string", false, "Assignment status."), query("source", "string", false, "Assignment source."), pageParam(), pageSizeParam(),
			}, nil, listOK("CourseAssignmentsListResponse")),
			postOp("Courses", "Admin create assignment", "Admin manually assigns a published course to an executor.", true, nil, body("CreateAssignmentRequest"), created("CourseAssignment")),
		),
		"/api/v1/my/course-assignments":                                      get("Courses", "My course assignments", "Executor paginated active course assignments.", true, []any{pageParam(), pageSizeParam()}, nil, listOK("CourseAssignmentsListResponse")),
		"/api/v1/my/course-assignments/{id}":                                 get("Courses", "My assignment details", "Executor reads one own course assignment with progress.", true, []any{path("id", "string", "Assignment UUID.")}, nil, ok("CourseAssignment")),
		"/api/v1/my/course-assignments/{id}/materials/{materialId}/complete": post("Courses", "Complete assignment material", "Executor marks one assignment material completed; assignment is completed automatically at 100%.", true, []any{path("id", "string", "Assignment UUID."), path("materialId", "string", "Material UUID.")}, nil, ok("CourseAssignment")),
		"/api/v1/my/course-assignments/{id}/mark-completed":                  post("Courses", "Mark assignment completed", "Backward-compatible shortcut that completes all current materials and updates assignment progress.", true, []any{path("id", "string", "Assignment UUID.")}, nil, ok("CourseAssignment")),

		"/api/v1/my/notifications":           get("Notifications", "My notifications", "Authenticated user notification list with filters.", true, []any{query("status", "string", false, "Notification status."), query("type", "string", false, "Notification type."), query("unread_only", "boolean", false, "Only unread notifications."), pageParam(), pageSizeParam()}, nil, listOK("NotificationsListResponse")),
		"/api/v1/my/notifications/{id}":      get("Notifications", "My notification details", "Authenticated user reads one own notification.", true, []any{path("id", "string", "Notification UUID.")}, nil, ok("Notification")),
		"/api/v1/my/notifications/{id}/read": post("Notifications", "Mark notification read", "Idempotently marks one notification as read.", true, []any{path("id", "string", "Notification UUID.")}, nil, ok("Notification")),
		"/api/v1/my/notifications/read-all":  post("Notifications", "Mark all notifications read", "Idempotently marks all current user notifications as read.", true, nil, nil, ok("MarkAllReadResponse")),
		"/api/v1/admin/notifications":        get("Notifications", "Admin notifications list", "Admin notification list with user, type, status and channel filters.", true, []any{query("user_id", "string", false, "User UUID."), query("type", "string", false, "Notification type."), query("status", "string", false, "Notification status."), query("channel", "string", false, "Notification channel."), pageParam(), pageSizeParam()}, nil, listOK("NotificationsListResponse")),
		"/api/v1/admin/notifications/{id}":   get("Notifications", "Admin notification details", "Admin reads any notification.", true, []any{path("id", "string", "Notification UUID.")}, nil, ok("Notification")),

		"/api/v1/my/chats": pathItem(
			getOp("Chats", "My chats", "Authenticated chat participant list.", true, []any{pageParam(), pageSizeParam()}, nil, listOK("ChatsListResponse")),
			postOp("Chats", "Create direct chat", "Creates or returns a one-to-one dialog with another active user.", true, nil, body("CreateDirectChatRequest"), created("ChatDetail")),
		),
		"/api/v1/my/chats/{id}": get("Chats", "My chat details", "Authenticated participant reads chat details.", true, []any{path("id", "string", "Chat UUID.")}, nil, ok("ChatDetail")),
		"/api/v1/my/chats/{id}/messages": pathItem(
			getOp("Chats", "My chat messages", "Participant reads chat messages oldest-first.", true, []any{path("id", "string", "Chat UUID."), pageParam(), pageSizeParam()}, nil, listOK("MessagesListResponse")),
			postOp("Chats", "Send chat message", "Client/executor participant sends a text message with optional uploaded file ids and notifies the other participant.", true, []any{path("id", "string", "Chat UUID.")}, body("SendMessageRequest"), created("Message")),
		),
		"/api/v1/my/chats/{id}/messages/{messageId}": pathItem(
			patchOp("Chats", "Edit chat message", "Message author updates text while keeping existing attachments.", true, []any{path("id", "string", "Chat UUID."), path("messageId", "string", "Message UUID.")}, body("UpdateMessageRequest"), ok("Message")),
			deleteOp("Chats", "Delete chat message", "Message author soft-deletes a message.", true, []any{path("id", "string", "Chat UUID."), path("messageId", "string", "Message UUID.")}, nil, ok("StatusResponse")),
		),
		"/api/v1/my/chats/{id}/read":        post("Chats", "Mark chat read", "Updates participant-level last_read_at.", true, []any{path("id", "string", "Chat UUID.")}, nil, ok("StatusResponse")),
		"/api/v1/admin/chats":               get("Chats", "Admin chats list", "Admin paginated chat list.", true, []any{pageParam(), pageSizeParam()}, nil, listOK("ChatsListResponse")),
		"/api/v1/admin/chats/{id}":          get("Chats", "Admin chat details", "Admin reads any chat details.", true, []any{path("id", "string", "Chat UUID.")}, nil, ok("ChatDetail")),
		"/api/v1/admin/chats/{id}/messages": get("Chats", "Admin chat messages", "Admin reads messages for any chat.", true, []any{path("id", "string", "Chat UUID."), pageParam(), pageSizeParam()}, nil, listOK("MessagesListResponse")),
	}
}

func components() map[string]any {
	return map[string]any{
		"securitySchemes": map[string]any{
			"bearerAuth": map[string]any{"type": "http", "scheme": "bearer", "bearerFormat": "JWT"},
		},
		"schemas": schemas(),
	}
}

func schemas() map[string]any {
	return map[string]any{
		"ErrorResponse": obj(
			req("error", ref("ErrorBody")),
		),
		"ErrorBody": obj(
			req("code", str("Machine-readable error code.")),
			req("message", str("Human-readable error message.")),
			prop("request_id", str("Request identifier from X-Request-ID.")),
		),
		"StatusResponse":       obj(prop("status", str("Operation status."))),
		"HealthResponse":       obj(prop("status", str("Service status."))),
		"PingResponse":         obj(prop("message", str("pong"))),
		"RegisterRequest":      obj(req("email", str("User email.")), req("password", str("Password, minimum 8 chars.")), req("role", enumStr("client", "coach")), prop("profile_name", str("Initial profile display/company name.")), prop("phone", str("Phone for client profile.")), prop("website", str("Website.")), prop("client_type", enumStr("too", "ip", "representative")), prop("tax_number", str("BIN/IIN.")), prop("contact_name", str("Contact person.")), prop("contact_position", str("Contact position.")), prop("address", str("Legal/contact address.")), prop("about", str("Profile description."))),
		"LoginRequest":         obj(req("email", str("User email.")), req("password", str("User password."))),
		"RefreshRequest":       obj(req("refresh_token", str("Refresh JWT."))),
		"LogoutRequest":        obj(req("refresh_token", str("Refresh JWT to revoke."))),
		"TokenPair":            obj(req("access_token", str("Access JWT.")), req("refresh_token", str("Refresh JWT."))),
		"AuthResponse":         obj(req("user_id", uuidStr()), req("email", str("User email.")), req("role", enumStr("client", "executor", "coach", "admin")), req("verification_status", verificationStatusSchema()), req("access_token", str("Access JWT.")), req("refresh_token", str("Refresh JWT."))),
		"MeResponse":           obj(req("id", uuidStr()), req("email", str("User email.")), req("role", enumStr("client", "executor", "coach", "admin")), req("verification_status", verificationStatusSchema()), req("profile", freeObj("Current role profile."))),
		"ProfileResponse":      freeObj("Role-specific profile payload."),
		"UpdateProfileRequest": freeObj("Role-specific profile fields: company_name, phone, about, display_name, bio, years_experience, expertise."),
		"SetAvatarRequest":     obj(req("upload_id", uuidStr())),

		"ExecutorLeadSubmittedResponse": obj(req("lead_id", uuidStr()), req("status", leadStatusSchema()), req("message", str("Human-readable submission message."))),
		"ExecutorLeadListResponse":      listSchema("ExecutorLeadView"),
		"ExecutorLeadView": obj(
			req("id", uuidStr()),
			req("email", str("Executor email.")),
			req("first_name", str("First name.")),
			req("last_name", str("Last name.")),
			prop("middle_name", str("Middle name.")),
			req("iin", str("Kazakhstan IIN, 12 digits.")),
			req("phone", str("Phone.")),
			req("city", str("City.")),
			req("experience_level", str("Experience level label.")),
			req("specializations", arr(str("Specialization."))),
			req("education", str("Education, certificates and courses.")),
			prop("work_format", str("Preferred work format.")),
			prop("hourly_rate", num("Hourly rate in KZT.")),
			req("about", str("About executor.")),
			req("terms_accepted", boolProp("Terms acceptance.")),
			req("status", leadStatusSchema()),
			req("priority", intProp("0 normal, 1 high, 2 urgent.")),
			prop("notes", str("Admin notes.")),
			prop("rejection_reason", str("Rejection reason.")),
			req("submitted_at", dateTime()),
			prop("reviewed_at", dateTime()),
			prop("reviewed_by", uuidStr()),
			prop("converted_at", dateTime()),
			prop("converted_user_id", uuidStr()),
			req("created_at", dateTime()),
			req("updated_at", dateTime()),
			prop("documents", arr(ref("ExecutorLeadDocument"))),
		),
		"ExecutorLeadDocument":    obj(req("id", uuidStr()), req("document_type", enumStr("identity", "education", "ip_registration", "other")), req("url", str("Public local URL.")), req("original_name", str("Original filename.")), req("mime_type", str("Detected MIME type.")), req("size_bytes", intProp("File size in bytes.")), req("created_at", dateTime())),
		"UpdateLeadStatusRequest": obj(req("status", enumStr("new", "in_review", "approved")), prop("notes", str("Admin notes."))),
		"ApproveLeadRequest":      obj(prop("notes", str("Admin notes."))),
		"RejectLeadRequest":       obj(req("reason", str("Rejection reason."))),
		"ApproveLeadResponse":     obj(req("status", str("converted")), req("user_id", uuidStr())),

		"UploadView": obj(
			req("id", uuidStr()),
			req("author_id", uuidStr()),
			req("file_path", str("Storage key relative to local uploads root.")),
			req("url", str("Public URL served by the backend.")),
			req("original_name", str("Original uploaded filename.")),
			req("mime_type", str("Detected MIME type.")),
			req("size_bytes", intProp("File size in bytes.")),
			req("created_at", dateTime()),
		),
		"UploadListResponse": listItemsOnlySchema("UploadView"),
		"AttachRequest": obj(
			req("upload_ids", arr(uuidStr())),
			req("target_type", targetTypeSchema()),
			req("target_id", uuidStr()),
			prop("metadata", freeObj("Optional attachment metadata.")),
		),
		"ReorderAttachmentsRequest": obj(req("ids", arr(uuidStr()))),
		"AttachmentView": obj(
			req("id", uuidStr()),
			req("upload_id", uuidStr()),
			req("target_type", targetTypeSchema()),
			req("target_id", uuidStr()),
			req("sort_order", intProp("Sort order inside target.")),
			req("metadata", freeObj("Attachment metadata.")),
			req("created_at", dateTime()),
			req("url", str("Public uploaded file URL.")),
			req("original_name", str("Original uploaded filename.")),
			req("mime_type", str("Detected MIME type.")),
			req("size_bytes", intProp("File size in bytes.")),
		),
		"AttachmentListResponse": listItemsOnlySchema("AttachmentView"),

		"CreditWalletRequest":  obj(req("amount", num("Amount to credit.")), prop("reason", str("Reason shown in wallet history."))),
		"Wallet":               obj(req("user_id", uuidStr()), req("balance", num("Internal balance.")), req("currency", str("Currency.")), req("updated_at", dateTime())),
		"WalletTransaction":    obj(req("id", uuidStr()), req("user_id", uuidStr()), req("amount", num("Amount.")), req("direction", enumStr("credit", "debit")), req("currency", str("Currency.")), req("reason", str("Reason.")), prop("order_id", uuidStr()), prop("created_by", uuidStr()), req("created_at", dateTime())),
		"WalletResponse":       obj(req("wallet", ref("Wallet")), req("transactions", arr(ref("WalletTransaction")))),
		"WalletCreditResponse": obj(req("wallet", ref("Wallet"))),

		"CreateOrderRequest":    obj(req("title", str("Order title.")), req("description", str("Order description.")), prop("category_id", intProp("Category id.")), prop("category_slug", str("Category slug.")), req("budget_amount", num("Budget/escrow amount.")), prop("currency", str("Three-letter currency code.")), prop("deadline_at", dateTime()), prop("region", str("Region or online.")), prop("promotions", arr(enumStr("top", "pin", "highlight")))),
		"UpdateOrderRequest":    obj(prop("title", str("Order title.")), prop("description", str("Order description.")), prop("category_id", intProp("Category id.")), prop("category_slug", str("Category slug.")), prop("budget_amount", num("Budget amount.")), prop("currency", str("Three-letter currency code.")), prop("deadline_at", dateTime()), prop("region", str("Region or online.")), prop("promotions", arr(enumStr("top", "pin", "highlight")))),
		"OrderResponse":         obj(req("id", uuidStr()), prop("client_id", uuidStr()), prop("category_id", intProp("Category id.")), prop("category_slug", str("Category slug.")), prop("category_name", str("Category name.")), req("title", str("Order title.")), req("description", str("Order description.")), req("budget_amount", num("Budget amount.")), req("currency", str("Currency.")), prop("deadline_at", dateTime()), prop("region", str("Region or online.")), req("promotions", arr(str("Promotion code."))), req("posting_fee", num("Posting fee.")), req("promotion_fee", num("Promotion fee.")), req("escrow_amount", num("Reserved budget.")), req("total_charge", num("Wallet debit total.")), req("payment_status", enumStr("unpaid", "paid", "refunded", "released")), prop("promoted_until", dateTime()), prop("pinned_until", dateTime()), prop("highlighted_until", dateTime()), req("status", enumStr("draft", "payment_pending", "published", "in_progress", "completed", "cancelled", "archived")), prop("published_at", dateTime()), prop("cancelled_at", dateTime()), req("created_at", dateTime()), req("updated_at", dateTime())),
		"OrdersListResponse":    listSchema("OrderResponse"),
		"OrderDetailsResponse":  obj(req("order", ref("OrderResponse")), prop("latest_payment", ref("PaymentTransaction"))),
		"PaymentTransaction":    obj(req("id", uuidStr()), prop("order_id", uuidStr()), prop("response_id", uuidStr()), req("provider", str("Provider name.")), prop("provider_ref", str("Provider transaction reference.")), req("amount", num("Amount.")), req("currency", str("Currency.")), req("status", enumStr("pending", "succeeded", "failed", "refunded", "cancelled")), req("initiated_at", dateTime())),
		"SubmitOrderResponse":   obj(req("order", ref("OrderResponse")), req("payment", ref("SubmitPaymentNextStep"))),
		"SubmitPaymentNextStep": obj(req("transaction_id", uuidStr()), req("provider", str("Provider.")), req("status", str("Provider/payment status.")), req("amount", num("Amount.")), req("currency", str("Currency.")), prop("checkout_url", str("Checkout URL.")), prop("provider_ref", str("Provider transaction reference."))),
		"CreatePaymentRequest":  obj(req("order_id", uuidStr()), req("amount", num("Expected total amount calculated by backend for the order."))),
		"CreatePaymentResponse": obj(req("transaction_id", uuidStr()), req("yookassa_payment_id", str("YooKassa payment id.")), req("status", str("Payment transaction status.")), req("amount", num("Amount.")), req("currency", str("Currency.")), req("confirmation_url", str("YooKassa redirect URL."))),
		"YooKassaWebhook":       obj(req("type", str("Webhook object type.")), req("event", str("YooKassa event, for example payment.succeeded.")), req("object", freeObj("YooKassa payment object."))),

		"CreateResponseRequest": obj(prop("cover_letter", str("Cover letter.")), prop("proposed_amount", num("Proposed amount.")), prop("currency", str("Currency."))),
		"UpdateResponseRequest": obj(prop("cover_letter", str("Cover letter.")), prop("proposed_amount", num("Proposed amount.")), prop("currency", str("Currency."))),
		"ResponseView":          obj(req("id", uuidStr()), req("order_id", uuidStr()), prop("executor_id", uuidStr()), prop("cover_letter", str("Cover letter.")), prop("proposed_amount", num("Proposed amount.")), req("currency", str("Currency.")), req("status", enumStr("draft", "payment_pending", "submitted", "accepted", "rejected", "cancelled")), req("is_paid", boolProp("Payment flag.")), prop("paid_at", dateTime()), req("created_at", dateTime()), req("updated_at", dateTime()), prop("order_title", str("Order title."))),
		"ResponsesListResponse": listSchema("ResponseView"),
		"SubmitResponsePayload": obj(req("response", ref("ResponseView")), req("payment", ref("SubmitPaymentNextStep"))),
		"Selection":             obj(req("order_id", uuidStr()), req("order_status", str("Order status.")), prop("selected_response_id", uuidStr()), prop("selected_executor_id", uuidStr()), prop("selected_response_status", str("Response status."))),
		"CreateReviewRequest":   obj(req("rating", intProp("Rating from 1 to 5.")), prop("comment", str("Review comment."))),
		"Review":                obj(req("id", uuidStr()), req("order_id", uuidStr()), req("client_id", uuidStr()), req("executor_id", uuidStr()), req("rating", intProp("Rating from 1 to 5.")), prop("comment", str("Review comment.")), req("created_at", dateTime()), req("updated_at", dateTime())),
		"ReviewsListResponse":   listSchema("Review"),
		"CreateEntityReviewRequest": obj(
			req("target_type", reviewTargetTypeSchema()),
			req("target_id", uuidStr()),
			req("rating", intProp("Rating from 1 to 5.")),
			prop("comment", str("Review comment.")),
			prop("metadata", freeObj("Optional review metadata.")),
		),
		"EntityReview": obj(
			req("id", uuidStr()),
			prop("author_id", uuidStr()),
			req("target_type", reviewTargetTypeSchema()),
			req("target_id", uuidStr()),
			req("rating", intProp("Rating from 1 to 5.")),
			prop("comment", str("Review comment.")),
			req("metadata", freeObj("Review metadata.")),
			req("created_at", dateTime()),
			req("updated_at", dateTime()),
		),
		"EntityReviewsListResponse": listSchema("EntityReview"),
		"EntityRatingSummary":       obj(req("target_type", reviewTargetTypeSchema()), req("target_id", uuidStr()), req("rating_avg", num("Average rating.")), req("rating_count", intProp("Total reviews.")), prop("updated_at", dateTime())),
		"RatingInfo":                obj(req("executor_id", uuidStr()), req("reviews_count_total", intProp("Total reviews.")), req("reviews_count_recent", intProp("Recent reviews counted.")), req("avg_rating_recent", num("Average rating in recent window.")), req("avg_rating_total", num("Total average rating."))),
		"Sanction":                  obj(req("id", uuidStr()), req("executor_id", uuidStr()), req("status", enumStr("active", "resolved", "expired")), req("reason", str("Sanction reason.")), req("severity", intProp("Severity 1..5.")), req("started_at", dateTime()), prop("ends_at", dateTime()), prop("expired_at", dateTime()), prop("resolved_at", dateTime())),
		"SanctionsResponse":         obj(req("items", arr(ref("Sanction")))),
		"AdminSanctionsResponse":    listSchema("Sanction"),
		"ExpireSanctionsResponse":   obj(req("expired_count", intProp("Number of active due sanctions marked expired."))),

		"CreateCourseRequest":           obj(req("title", str("Course title.")), prop("subtitle", str("Short course subtitle.")), prop("description", str("Course description.")), prop("slug", str("Course slug.")), prop("category", str("Course category.")), prop("level", enumStr("beginner", "intermediate", "advanced")), prop("language", str("Two-letter language code.")), prop("price", num("Course price.")), prop("currency", str("Currency.")), prop("duration_minutes", intProp("Estimated duration.")), prop("cover_upload_id", uuidStr()), prop("cover_url", str("Cover URL.")), prop("tags", arr(str("Course tag."))), prop("learning_outcomes", arr(str("Learning outcome."))), prop("requirements", arr(str("Requirement."))), prop("certificate_enabled", boolProp("Certificate availability."))),
		"UpdateCourseRequest":           obj(prop("title", str("Course title.")), prop("subtitle", str("Short course subtitle.")), prop("description", str("Course description.")), prop("slug", str("Course slug.")), prop("category", str("Course category.")), prop("level", enumStr("beginner", "intermediate", "advanced")), prop("language", str("Two-letter language code.")), prop("price", num("Course price.")), prop("currency", str("Currency.")), prop("duration_minutes", intProp("Estimated duration.")), prop("cover_upload_id", uuidStr()), prop("cover_url", str("Cover URL.")), prop("tags", arr(str("Course tag."))), prop("learning_outcomes", arr(str("Learning outcome."))), prop("requirements", arr(str("Requirement."))), prop("certificate_enabled", boolProp("Certificate availability."))),
		"Course":                        obj(req("id", uuidStr()), prop("coach_id", uuidStr()), prop("created_by", uuidStr()), req("title", str("Course title.")), prop("subtitle", str("Course subtitle.")), prop("description", str("Course description.")), prop("slug", str("Course slug.")), prop("category", str("Course category.")), req("level", enumStr("beginner", "intermediate", "advanced")), req("language", str("Language code.")), req("price", num("Course price.")), req("currency", str("Currency.")), req("duration_minutes", intProp("Estimated duration.")), prop("cover_upload_id", uuidStr()), prop("cover_url", str("Cover URL.")), req("tags", arr(str("Course tag."))), req("learning_outcomes", arr(str("Learning outcome."))), req("requirements", arr(str("Requirement."))), req("certificate_enabled", boolProp("Certificate availability.")), req("status", enumStr("draft", "published", "archived")), req("moderation_status", enumStr("draft", "in_review", "approved", "rejected")), req("enrollment_count", intProp("Enrollment count.")), req("rating_avg", num("Average rating.")), req("rating_count", intProp("Review count.")), prop("published_at", dateTime()), prop("archived_at", dateTime()), req("created_at", dateTime()), req("updated_at", dateTime())),
		"CoursesListResponse":           listSchema("Course"),
		"CourseDetailResponse":          obj(req("course", ref("Course")), req("materials", arr(ref("CourseMaterial")))),
		"CreateMaterialRequest":         obj(req("title", str("Material title.")), prop("description", str("Material description.")), req("type", enumStr("video", "pdf", "link", "text")), prop("upload_id", uuidStr()), prop("url", str("URL for video/pdf/link.")), prop("content", str("Text content for text material.")), prop("position", intProp("Sort order.")), prop("duration_seconds", intProp("Video/material duration.")), prop("is_preview", boolProp("Preview availability."))),
		"UpdateMaterialRequest":         obj(prop("title", str("Material title.")), prop("description", str("Material description.")), prop("type", enumStr("video", "pdf", "link", "text")), prop("upload_id", uuidStr()), prop("url", str("URL for video/pdf/link.")), prop("content", str("Text content for text material.")), prop("position", intProp("Sort order.")), prop("duration_seconds", intProp("Video/material duration.")), prop("is_preview", boolProp("Preview availability."))),
		"CourseMaterial":                obj(req("id", uuidStr()), req("course_id", uuidStr()), req("title", str("Material title.")), prop("description", str("Material description.")), req("type", enumStr("video", "pdf", "link", "text")), prop("upload_id", uuidStr()), prop("url", str("Material URL.")), prop("content", str("Material content.")), req("position", intProp("Sort order.")), req("duration_seconds", intProp("Video/material duration.")), req("is_preview", boolProp("Preview availability.")), req("created_at", dateTime()), req("updated_at", dateTime())),
		"CreatorAnalytics":              obj(req("total_courses", intProp("Total owned courses.")), req("published_courses", intProp("Published courses.")), req("draft_courses", intProp("Draft courses.")), req("archived_courses", intProp("Archived courses.")), req("total_materials", intProp("Total materials.")), req("total_assignments", intProp("Total assignments.")), req("active_students", intProp("Active students.")), req("completed_assignments", intProp("Completed assignments.")), req("average_progress", num("Average progress.")), req("executor_can_create", boolProp("Current executor creator eligibility.")), prop("executor_min_rating", num("Required rating for executor creators.")), prop("executor_min_review_count", intProp("Required review count for executor creators."))),
		"CourseStudent":                 obj(req("assignment_id", uuidStr()), req("course_id", uuidStr()), req("executor_id", uuidStr()), prop("executor_name", str("Executor display name.")), prop("executor_email", str("Executor email.")), req("status", enumStr("assigned", "in_progress", "completed", "overdue", "cancelled")), req("progress_percent", intProp("Progress percent.")), req("completed_materials", intProp("Completed materials.")), req("total_materials", intProp("Total materials.")), req("assigned_at", dateTime()), prop("due_at", dateTime()), prop("completed_at", dateTime()), prop("last_activity_at", dateTime())),
		"CourseStudentsListResponse":    listSchema("CourseStudent"),
		"CreateAssignmentRequest":       obj(req("course_id", uuidStr()), req("executor_id", uuidStr()), req("source", enumStr("manual_admin", "sanction_low_rating_first", "sanction_low_rating_repeat")), prop("reason", str("Assignment reason.")), prop("due_at", dateTime())),
		"CourseProgress":                obj(prop("id", uuidStr()), req("assignment_id", uuidStr()), req("executor_id", uuidStr()), req("progress_percent", intProp("Progress percent 0..100.")), req("status", enumStr("assigned", "in_progress", "completed", "overdue", "cancelled")), req("completed_materials", intProp("Completed material count.")), req("total_materials", intProp("Total material count.")), prop("completed_material_ids", arr(uuidStr())), prop("last_activity_at", dateTime()), prop("completed_at", dateTime()), prop("created_at", dateTime()), prop("updated_at", dateTime())),
		"CourseAssignment":              obj(req("id", uuidStr()), req("course_id", uuidStr()), req("executor_id", uuidStr()), prop("sanction_id", uuidStr()), prop("assigned_by", uuidStr()), prop("reason", str("Reason.")), req("source", str("Source.")), req("status", enumStr("assigned", "in_progress", "completed", "overdue", "cancelled")), req("assigned_at", dateTime()), prop("due_at", dateTime()), prop("completed_at", dateTime()), req("created_at", dateTime()), req("updated_at", dateTime()), prop("progress", ref("CourseProgress"))),
		"CourseAssignmentsListResponse": listSchema("CourseAssignment"),

		"Notification":              obj(req("id", uuidStr()), req("user_id", uuidStr()), req("type", str("Notification type.")), req("channel", enumStr("in_app", "email", "sms")), req("status", enumStr("pending", "sent", "failed", "read")), req("payload", freeObj("Machine-readable payload.")), req("created_at", dateTime()), prop("sent_at", dateTime()), prop("read_at", dateTime()), prop("error_message", str("Delivery error."))),
		"NotificationsListResponse": listSchema("Notification"),
		"MarkAllReadResponse":       obj(req("updated", intProp("Number of rows updated."))),
		"ChatParticipant":           obj(req("user_id", uuidStr()), req("joined_at", dateTime()), prop("last_read_at", dateTime())),
		"ChatSummary":               obj(req("chat_id", uuidStr()), req("chat_type", enumStr("order", "direct")), prop("order_id", uuidStr()), prop("user_a_id", uuidStr()), prop("user_b_id", uuidStr()), req("participants", arr(ref("ChatParticipant"))), prop("last_message_preview", str("Last message preview.")), prop("last_message_at", dateTime()), req("unread_count", intProp("Unread message count.")), req("has_unread", boolProp("Has unread messages."))),
		"ChatDetail":                obj(req("chat_id", uuidStr()), req("chat_type", enumStr("order", "direct")), prop("order_id", uuidStr()), prop("order_status", str("Order status for order chats.")), prop("client_id", uuidStr()), prop("selected_executor_id", uuidStr()), prop("user_a_id", uuidStr()), prop("user_b_id", uuidStr()), req("participants", arr(ref("ChatParticipant"))), prop("last_message_at", dateTime())),
		"ChatsListResponse":         listSchema("ChatSummary"),
		"MessageAttachment":         obj(req("id", uuidStr()), req("upload_id", uuidStr()), req("file_path", str("Local storage path.")), prop("url", str("Public local URL.")), req("original_name", str("Original file name.")), req("mime_type", str("MIME type.")), req("size_bytes", intProp("File size in bytes.")), req("created_at", dateTime())),
		"Message":                   obj(req("id", uuidStr()), req("chat_id", uuidStr()), prop("sender_user_id", uuidStr()), req("sender_type", enumStr("user", "system")), req("text", str("Message text.")), req("attachments", arr(ref("MessageAttachment"))), req("created_at", dateTime()), prop("edited_at", dateTime()), prop("deleted_at", dateTime())),
		"MessagesListResponse":      obj(req("items", arr(ref("Message"))), req("page", intProp("Page number.")), req("page_size", intProp("Page size.")), req("total", intProp("Total matching items.")), req("order", enumStr("asc"))),
		"CreateDirectChatRequest": obj(
			req("participant_id", uuidStr()),
		),
		"SendMessageRequest": obj(
			prop("text", str("Message text. Required when attachment_ids is empty.")),
			prop("attachment_ids", arr(uuidStr())),
		),
		"UpdateMessageRequest": obj(req("text", str("Updated message text."))),
	}
}

func tag(name, description string) map[string]any {
	return map[string]any{"name": name, "description": description}
}

func pathItem(ops ...map[string]any) map[string]any {
	out := map[string]any{}
	for _, op := range ops {
		for method, body := range op {
			out[method] = body
		}
	}
	return out
}

func get(tagName, summary, description string, secured bool, params []any, requestBody map[string]any, responses map[string]any) map[string]any {
	return pathItem(getOp(tagName, summary, description, secured, params, requestBody, responses))
}

func post(tagName, summary, description string, secured bool, params []any, requestBody map[string]any, responses map[string]any) map[string]any {
	return pathItem(postOp(tagName, summary, description, secured, params, requestBody, responses))
}

func getOp(tagName, summary, description string, secured bool, params []any, requestBody map[string]any, responses map[string]any) map[string]any {
	return method("get", tagName, summary, description, secured, params, requestBody, responses)
}

func postOp(tagName, summary, description string, secured bool, params []any, requestBody map[string]any, responses map[string]any) map[string]any {
	return method("post", tagName, summary, description, secured, params, requestBody, responses)
}

func patchOp(tagName, summary, description string, secured bool, params []any, requestBody map[string]any, responses map[string]any) map[string]any {
	return method("patch", tagName, summary, description, secured, params, requestBody, responses)
}

func deleteOp(tagName, summary, description string, secured bool, params []any, requestBody map[string]any, responses map[string]any) map[string]any {
	return method("delete", tagName, summary, description, secured, params, requestBody, responses)
}

func method(name, tagName, summary, description string, secured bool, params []any, requestBody map[string]any, responses map[string]any) map[string]any {
	op := map[string]any{
		"tags":        []any{tagName},
		"summary":     summary,
		"description": description,
		"operationId": operationID(name, tagName, summary),
		"responses":   withDefaultErrors(responses),
	}
	if secured {
		op["security"] = []any{map[string]any{"bearerAuth": []any{}}}
	}
	if len(params) > 0 {
		op["parameters"] = params
	}
	if requestBody != nil {
		op["requestBody"] = requestBody
	}
	return map[string]any{name: op}
}

func withDefaultErrors(responses map[string]any) map[string]any {
	if responses == nil {
		responses = map[string]any{}
	}
	responses["400"] = responseRef("Bad request.", "ErrorResponse")
	responses["401"] = responseRef("Unauthorized.", "ErrorResponse")
	responses["403"] = responseRef("Forbidden.", "ErrorResponse")
	responses["404"] = responseRef("Not found.", "ErrorResponse")
	responses["409"] = responseRef("Conflict.", "ErrorResponse")
	responses["500"] = responseRef("Internal server error.", "ErrorResponse")
	return responses
}

func ok(schema string) map[string]any {
	return map[string]any{"200": responseRef("Successful response.", schema)}
}

func created(schema string) map[string]any {
	return map[string]any{"201": responseRef("Created.", schema)}
}

func listOK(schema string) map[string]any {
	return map[string]any{"200": responseRef("Paginated list.", schema)}
}

func textOK() map[string]any {
	return map[string]any{"200": map[string]any{"description": "Plain text metrics output.", "content": map[string]any{"text/plain": map[string]any{"schema": map[string]any{"type": "string"}}}}}
}

func responseRef(description, schema string) map[string]any {
	return map[string]any{
		"description": description,
		"content": map[string]any{
			"application/json": map[string]any{"schema": ref(schema)},
		},
	}
}

func body(schema string) map[string]any {
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"application/json": map[string]any{"schema": ref(schema)},
		},
	}
}

func multipartFilesBody() map[string]any {
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"multipart/form-data": map[string]any{
				"schema": obj(req("file", arr(map[string]any{"type": "string", "format": "binary"}))),
			},
		},
	}
}

func executorLeadMultipartBody() map[string]any {
	binary := map[string]any{"type": "string", "format": "binary"}
	return map[string]any{
		"required": true,
		"content": map[string]any{
			"multipart/form-data": map[string]any{
				"schema": obj(
					req("email", str("Executor email.")),
					req("password", str("Password.")),
					req("first_name", str("First name.")),
					req("last_name", str("Last name.")),
					prop("middle_name", str("Middle name.")),
					req("iin", str("Kazakhstan IIN, 12 digits.")),
					req("phone", str("Phone.")),
					req("city", str("City.")),
					req("experience_level", str("Experience level, for example 3-5.")),
					req("specializations", arr(str("Specialization values or send as JSON array string."))),
					req("education", str("Education and certificates.")),
					prop("work_format", str("Preferred work format.")),
					prop("hourly_rate", num("Hourly rate in KZT.")),
					req("about", str("About executor.")),
					req("terms_accepted", boolProp("Must be true.")),
					req("identity_document", binary),
					req("education_document", binary),
					prop("ip_registration_document", binary),
				),
			},
		},
	}
}

func path(name, schemaType, description string) map[string]any {
	return param(name, "path", schemaType, true, description)
}

func query(name, schemaType string, required bool, description string) map[string]any {
	return param(name, "query", schemaType, required, description)
}

func pageParam() map[string]any {
	return query("page", "integer", false, "Page number, defaults to 1.")
}

func pageSizeParam() map[string]any {
	return query("page_size", "integer", false, "Page size, defaults to 20 and is capped at 100.")
}

func param(name, in, schemaType string, required bool, description string) map[string]any {
	schema := map[string]any{"type": schemaType}
	if schemaType == "string" && strings.Contains(strings.ToUpper(description), "UUID") {
		schema["format"] = "uuid"
	}

	return map[string]any{
		"name":        name,
		"in":          in,
		"required":    required,
		"description": description,
		"schema":      schema,
	}
}

func operationID(methodName, tagName, summary string) string {
	words := identifierWords(methodName + " " + tagName + " " + summary)
	if len(words) == 0 {
		return methodName
	}

	var b strings.Builder
	b.WriteString(words[0])
	for _, word := range words[1:] {
		b.WriteString(titleWord(word))
	}
	return b.String()
}

func identifierWords(value string) []string {
	words := make([]string, 0)
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		words = append(words, b.String())
		b.Reset()
	}

	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		flush()
	}
	flush()
	return words
}

func titleWord(word string) string {
	runes := []rune(word)
	if len(runes) == 0 {
		return ""
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func obj(props ...map[string]any) map[string]any {
	properties := map[string]any{}
	required := make([]any, 0)
	for _, p := range props {
		name, _ := p["name"].(string)
		schema := p["schema"]
		properties[name] = schema
		if isRequired, _ := p["required"].(bool); isRequired {
			required = append(required, name)
		}
	}
	out := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		out["required"] = required
	}
	return out
}

func prop(name string, schema map[string]any) map[string]any {
	return map[string]any{"name": name, "schema": schema}
}

func req(name string, schema map[string]any) map[string]any {
	return map[string]any{"name": name, "schema": schema, "required": true}
}

func ref(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

func str(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func uuidStr() map[string]any {
	return map[string]any{"type": "string", "format": "uuid"}
}

func dateTime() map[string]any {
	return map[string]any{"type": "string", "format": "date-time"}
}

func enumStr(values ...string) map[string]any {
	enum := make([]any, 0, len(values))
	for _, value := range values {
		enum = append(enum, value)
	}
	return map[string]any{"type": "string", "enum": enum}
}

func targetTypeSchema() map[string]any {
	return enumStr("profile_document", "order_attachment", "response_attachment", "review_attachment", "chat_attachment", "course_material")
}

func reviewTargetTypeSchema() map[string]any {
	return enumStr("user", "client", "executor", "coach", "profile", "order", "response", "review", "course", "course_material")
}

func verificationStatusSchema() map[string]any {
	return enumStr("none", "pending", "in_review", "verified", "rejected")
}

func leadStatusSchema() map[string]any {
	return enumStr("new", "in_review", "approved", "rejected", "converted")
}

func intProp(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
}

func num(description string) map[string]any {
	return map[string]any{"type": "number", "format": "double", "description": description}
}

func boolProp(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}

func arr(items map[string]any) map[string]any {
	return map[string]any{"type": "array", "items": items}
}

func freeObj(description string) map[string]any {
	return map[string]any{"type": "object", "additionalProperties": true, "description": description}
}

func listSchema(itemSchema string) map[string]any {
	return obj(
		req("items", arr(ref(itemSchema))),
		req("page", intProp("Page number.")),
		req("page_size", intProp("Page size.")),
		req("total", intProp("Total matching items.")),
	)
}

func listItemsOnlySchema(itemSchema string) map[string]any {
	return obj(req("items", arr(ref(itemSchema))))
}
