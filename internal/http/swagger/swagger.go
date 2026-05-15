package swagger

import (
	"net/http"

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
			tag("Executor Leads", "Public executor registration with document verification."),
			tag("Admin Executor Leads", "Admin review and conversion of executor leads."),
			tag("Orders", "Client order creation, management and public feed."),
			tag("Responses", "Executor paid responses and client response review."),
			tag("Dev Payments", "Development-only payment confirmation/failure endpoints."),
			tag("Selection & Lifecycle", "Client selection, completion and reopen flow."),
			tag("Reviews", "Order review and public executor review history."),
			tag("Rating & Sanctions", "Executor rating and sanction administration."),
			tag("Courses", "Coach/admin course management and executor assignments."),
			tag("Notifications", "In-app notification read models."),
			tag("Chats", "REST chat access for participants and admins."),
			tag("Admin Debug", "Admin read/debug endpoints."),
		},
		"paths":      paths(),
		"components": components(),
	}
}

func paths() map[string]any {
	return map[string]any{
		"/healthz":     get("System", "Health check", "Returns liveness status without requiring database access.", false, nil, nil, ok("HealthResponse")),
		"/readyz":      get("System", "Readiness check", "Checks database connectivity and reports whether the API is ready to serve traffic.", false, nil, nil, ok("HealthResponse")),
		"/metrics":     get("System", "Prometheus metrics", "Prometheus scrape endpoint when METRICS_ENABLED=true.", false, nil, nil, textOK()),
		"/api/v1/ping": get("System", "API ping", "Versioned API ping endpoint.", false, nil, nil, ok("PingResponse")),

		"/api/v1/auth/register": post("Auth", "Register user", "Creates a client or coach user and its role profile in one transaction. Executors must use the executor lead endpoint with documents.", false, nil, body("RegisterRequest"), created("AuthResponse")),
		"/api/v1/auth/login":    post("Auth", "Login", "Authenticates by email and password and returns access/refresh tokens.", false, nil, body("LoginRequest"), ok("AuthResponse")),
		"/api/v1/auth/refresh":  post("Auth", "Refresh tokens", "Rotates a valid refresh token and returns a new token pair.", false, nil, body("RefreshRequest"), ok("TokenPair")),
		"/api/v1/auth/logout":   post("Auth", "Logout", "Revokes the current refresh token.", true, nil, body("LogoutRequest"), ok("StatusResponse")),
		"/api/v1/auth/me":       get("Auth", "Current user", "Returns the authenticated user.", true, nil, nil, ok("MeResponse")),

		"/api/v1/profile": pathItem(
			getOp("Profile", "Get current profile", "Returns the profile table matching the current user's role.", true, nil, nil, ok("ProfileResponse")),
			patchOp("Profile", "Update current profile", "Partially updates the current role profile.", true, nil, body("UpdateProfileRequest"), ok("StatusResponse")),
		),
		"/api/v1/profile/avatar": pathItem(
			patchOp("Profile", "Set profile avatar", "Links one uploaded file as the current role profile avatar.", true, nil, body("SetAvatarRequest"), ok("StatusResponse")),
			deleteOp("Profile", "Clear profile avatar", "Removes the avatar link from the current role profile without deleting the uploaded file.", true, nil, nil, ok("StatusResponse")),
		),

		"/api/v1/files":      post("Uploads", "Upload files", "Uploads one or more files to local storage. Send multipart/form-data with repeated file fields.", true, nil, multipartFilesBody(), created("UploadListResponse")),
		"/api/v1/files/{id}": pathItem(getOp("Uploads", "Get uploaded file metadata", "Returns upload metadata and public local URL.", false, []any{path("id", "string", "Upload UUID.")}, nil, ok("UploadView")), deleteOp("Uploads", "Delete uploaded file", "Deletes an owned upload record and local file; admin can delete any upload.", true, []any{path("id", "string", "Upload UUID.")}, nil, ok("StatusResponse"))),
		"/api/v1/my/files":   get("Uploads", "My uploaded files", "Lists uploads owned by the authenticated user.", true, nil, nil, ok("UploadListResponse")),

		"/api/v1/attachments": pathItem(
			getOp("Attachments", "List attachments", "Lists files attached to a target entity by target_type and target_id.", false, []any{query("target_type", "string", true, "Attachment target type."), query("target_id", "string", true, "Target UUID.")}, nil, ok("AttachmentListResponse")),
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
				query("deadline_before", "string", false, "Reserved RFC3339 deadline filter; current schema has no deadline column."),
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
		"/api/v1/executors/{executorId}/reviews": get("Reviews", "Executor reviews", "Public paginated reviews for an executor, newest first.", false, []any{path("executorId", "string", "Executor user UUID."), pageParam(), pageSizeParam()}, nil, listOK("ReviewsListResponse")),
		"/api/v1/executors/{executorId}/rating":  get("Rating & Sanctions", "Executor rating", "Public rating summary for an executor.", false, []any{path("executorId", "string", "Executor user UUID.")}, nil, ok("RatingInfo")),
		"/api/v1/my/sanctions":                   get("Rating & Sanctions", "My sanctions", "Executor/admin list of sanctions for the current user.", true, nil, nil, ok("SanctionsResponse")),
		"/api/v1/admin/sanctions":                get("Rating & Sanctions", "Admin list sanctions", "Admin paginated sanction list.", true, []any{pageParam(), pageSizeParam()}, nil, listOK("AdminSanctionsResponse")),
		"/api/v1/admin/sanctions/{id}":           get("Rating & Sanctions", "Admin get sanction", "Admin sanction details.", true, []any{path("id", "string", "Sanction UUID.")}, nil, ok("Sanction")),
		"/api/v1/admin/sanctions/{id}/lift":      post("Rating & Sanctions", "Admin lift sanction", "Marks an active sanction as resolved.", true, []any{path("id", "string", "Sanction UUID.")}, nil, ok("StatusResponse")),

		"/api/v1/coach/courses": pathItem(
			getOp("Courses", "Coach list courses", "Coach/admin paginated course list.", true, []any{query("status", "string", false, "Optional course status."), pageParam(), pageSizeParam()}, nil, listOK("CoursesListResponse")),
			postOp("Courses", "Create course", "Coach/admin creates a draft course.", true, nil, body("CreateCourseRequest"), created("Course")),
		),
		"/api/v1/coach/courses/{id}": pathItem(
			getOp("Courses", "Coach get course", "Coach/admin reads a course and its materials.", true, []any{path("id", "string", "Course UUID.")}, nil, ok("CourseDetailResponse")),
			patchOp("Courses", "Patch course", "Coach/admin updates a course title/description.", true, []any{path("id", "string", "Course UUID.")}, body("UpdateCourseRequest"), ok("Course")),
		),
		"/api/v1/coach/courses/{id}/publish":   post("Courses", "Publish course", "Moves a draft course to published.", true, []any{path("id", "string", "Course UUID.")}, nil, ok("Course")),
		"/api/v1/coach/courses/{id}/archive":   post("Courses", "Archive course", "Moves a published course to archived.", true, []any{path("id", "string", "Course UUID.")}, nil, ok("Course")),
		"/api/v1/coach/courses/{id}/materials": post("Courses", "Create material", "Adds a material to a course. video/pdf/link require url; text requires content.", true, []any{path("id", "string", "Course UUID.")}, body("CreateMaterialRequest"), created("CourseMaterial")),
		"/api/v1/coach/courses/{id}/materials/{materialId}": pathItem(
			patchOp("Courses", "Patch material", "Updates a course material.", true, []any{path("id", "string", "Course UUID."), path("materialId", "string", "Material UUID.")}, body("UpdateMaterialRequest"), ok("CourseMaterial")),
			deleteOp("Courses", "Delete material", "Deletes a course material.", true, []any{path("id", "string", "Course UUID."), path("materialId", "string", "Material UUID.")}, nil, ok("StatusResponse")),
		),
		"/api/v1/courses":      get("Courses", "Course catalog", "Authenticated executor/coach/admin course catalog. Executors only see published courses.", true, []any{query("status", "string", false, "Optional course status; executor is forced to published."), pageParam(), pageSizeParam()}, nil, listOK("CoursesListResponse")),
		"/api/v1/courses/{id}": get("Courses", "Catalog course details", "Authenticated course details and materials.", true, []any{path("id", "string", "Course UUID.")}, nil, ok("CourseDetailResponse")),
		"/api/v1/admin/course-assignments": pathItem(
			getOp("Courses", "Admin list course assignments", "Admin list with executor, course, status and source filters.", true, []any{
				query("executor_id", "string", false, "Executor user UUID."), query("course_id", "string", false, "Course UUID."),
				query("status", "string", false, "Assignment status."), query("source", "string", false, "Assignment source."), pageParam(), pageSizeParam(),
			}, nil, listOK("CourseAssignmentsListResponse")),
			postOp("Courses", "Admin create assignment", "Admin manually assigns a published course to an executor.", true, nil, body("CreateAssignmentRequest"), created("CourseAssignment")),
		),
		"/api/v1/my/course-assignments":                     get("Courses", "My course assignments", "Executor paginated active course assignments.", true, []any{pageParam(), pageSizeParam()}, nil, listOK("CourseAssignmentsListResponse")),
		"/api/v1/my/course-assignments/{id}":                get("Courses", "My assignment details", "Executor reads one own course assignment.", true, []any{path("id", "string", "Assignment UUID.")}, nil, ok("CourseAssignment")),
		"/api/v1/my/course-assignments/{id}/mark-completed": post("Courses", "Mark assignment completed", "Executor marks an assignment completed and notifies the assigning admin when present.", true, []any{path("id", "string", "Assignment UUID.")}, nil, ok("CourseAssignment")),

		"/api/v1/my/notifications":           get("Notifications", "My notifications", "Authenticated user notification list with filters.", true, []any{query("status", "string", false, "Notification status."), query("type", "string", false, "Notification type."), query("unread_only", "boolean", false, "Only unread notifications."), pageParam(), pageSizeParam()}, nil, listOK("NotificationsListResponse")),
		"/api/v1/my/notifications/{id}":      get("Notifications", "My notification details", "Authenticated user reads one own notification.", true, []any{path("id", "string", "Notification UUID.")}, nil, ok("Notification")),
		"/api/v1/my/notifications/{id}/read": post("Notifications", "Mark notification read", "Idempotently marks one notification as read.", true, []any{path("id", "string", "Notification UUID.")}, nil, ok("Notification")),
		"/api/v1/my/notifications/read-all":  post("Notifications", "Mark all notifications read", "Idempotently marks all current user notifications as read.", true, nil, nil, ok("MarkAllReadResponse")),
		"/api/v1/admin/notifications":        get("Notifications", "Admin notifications list", "Admin notification list with user, type, status and channel filters.", true, []any{query("user_id", "string", false, "User UUID."), query("type", "string", false, "Notification type."), query("status", "string", false, "Notification status."), query("channel", "string", false, "Notification channel."), pageParam(), pageSizeParam()}, nil, listOK("NotificationsListResponse")),
		"/api/v1/admin/notifications/{id}":   get("Notifications", "Admin notification details", "Admin reads any notification.", true, []any{path("id", "string", "Notification UUID.")}, nil, ok("Notification")),

		"/api/v1/my/chats":      get("Chats", "My chats", "Authenticated chat participant list.", true, []any{pageParam(), pageSizeParam()}, nil, listOK("ChatsListResponse")),
		"/api/v1/my/chats/{id}": get("Chats", "My chat details", "Authenticated participant reads chat details.", true, []any{path("id", "string", "Chat UUID.")}, nil, ok("ChatDetail")),
		"/api/v1/my/chats/{id}/messages": pathItem(
			getOp("Chats", "My chat messages", "Participant reads chat messages oldest-first.", true, []any{path("id", "string", "Chat UUID."), pageParam(), pageSizeParam()}, nil, listOK("MessagesListResponse")),
			postOp("Chats", "Send chat message", "Client/executor participant sends a text message and notifies the other participant.", true, []any{path("id", "string", "Chat UUID.")}, body("SendMessageRequest"), created("Message")),
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
			req("success", boolProp("Always false for intercepted debug errors.")),
			req("error", ref("ErrorBody")),
			req("request_id", str("Request identifier from X-Request-ID.")),
			req("timestamp", dateTime()),
			req("path", str("Request path.")),
			req("method", str("HTTP method.")),
			req("status", intProp("HTTP status code.")),
		),
		"ErrorBody": obj(
			req("code", str("Machine-readable code, for example HTTP_500.")),
			req("message", str("HTTP status text.")),
			req("error_trace", str("Original intercepted response body or panic value.")),
			req("stack_trace", str("runtime/debug.Stack output.")),
		),
		"StatusResponse":       obj(prop("status", str("Operation status."))),
		"HealthResponse":       obj(prop("status", str("Service status."))),
		"PingResponse":         obj(prop("message", str("pong"))),
		"RegisterRequest":      obj(req("email", str("User email.")), req("password", str("Password, minimum 8 chars.")), req("role", enumStr("client", "coach")), req("profile_name", str("Initial profile display/company name.")), prop("phone", str("Phone for client profile.")), prop("client_type", enumStr("too", "ip", "representative")), prop("tax_number", str("BIN/IIN.")), prop("contact_name", str("Contact person.")), prop("contact_position", str("Contact position.")), prop("address", str("Legal/contact address.")), prop("about", str("Profile description."))),
		"LoginRequest":         obj(req("email", str("User email.")), req("password", str("User password."))),
		"RefreshRequest":       obj(req("refresh_token", str("Refresh JWT."))),
		"LogoutRequest":        obj(req("refresh_token", str("Refresh JWT to revoke."))),
		"TokenPair":            obj(req("access_token", str("Access JWT.")), req("refresh_token", str("Refresh JWT."))),
		"AuthResponse":         obj(req("user", ref("User")), req("tokens", ref("TokenPair")), prop("profile", freeObj("Current role profile."))),
		"MeResponse":           obj(req("user", ref("User")), prop("profile", freeObj("Current role profile."))),
		"User":                 obj(req("id", uuidStr()), req("email", str("Email.")), req("role", enumStr("client", "executor", "coach", "admin")), req("is_active", boolProp("Active flag.")), prop("verification_status", verificationStatusSchema()), req("created_at", dateTime())),
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

		"CreateOrderRequest":    obj(req("title", str("Order title.")), req("description", str("Order description.")), prop("category_id", intProp("Category id.")), prop("category_slug", str("Category slug.")), req("budget_amount", num("Budget amount.")), prop("currency", str("Three-letter currency code.")), prop("promotions", arr(str("Promotion code.")))),
		"UpdateOrderRequest":    obj(prop("title", str("Order title.")), prop("description", str("Order description.")), prop("category_id", intProp("Category id.")), prop("category_slug", str("Category slug.")), prop("budget_amount", num("Budget amount.")), prop("currency", str("Three-letter currency code."))),
		"OrderResponse":         obj(req("id", uuidStr()), prop("client_id", uuidStr()), prop("category_id", intProp("Category id.")), prop("category_slug", str("Category slug.")), prop("category_name", str("Category name.")), req("title", str("Order title.")), req("description", str("Order description.")), req("budget_amount", num("Budget amount.")), req("currency", str("Currency.")), req("status", enumStr("draft", "payment_pending", "published", "in_progress", "completed", "cancelled", "archived")), prop("published_at", dateTime()), prop("cancelled_at", dateTime()), req("created_at", dateTime()), req("updated_at", dateTime())),
		"OrdersListResponse":    listSchema("OrderResponse"),
		"OrderDetailsResponse":  obj(req("order", ref("OrderResponse")), prop("latest_payment", ref("PaymentTransaction"))),
		"PaymentTransaction":    obj(req("id", uuidStr()), prop("order_id", uuidStr()), prop("response_id", uuidStr()), req("provider", str("Provider name.")), prop("provider_ref", str("Provider transaction reference.")), req("amount", num("Amount.")), req("currency", str("Currency.")), req("status", enumStr("pending", "succeeded", "failed", "refunded", "cancelled")), req("initiated_at", dateTime())),
		"SubmitOrderResponse":   obj(req("order", ref("OrderResponse")), req("payment", ref("SubmitPaymentNextStep"))),
		"SubmitPaymentNextStep": obj(req("transaction_id", uuidStr()), req("provider", str("Provider.")), req("status", str("Provider/payment status.")), req("amount", num("Amount.")), req("currency", str("Currency.")), prop("checkout_url", str("Checkout URL.")), prop("provider_ref", str("Provider transaction reference."))),

		"CreateResponseRequest":  obj(prop("cover_letter", str("Cover letter.")), prop("proposed_amount", num("Proposed amount.")), prop("currency", str("Currency."))),
		"UpdateResponseRequest":  obj(prop("cover_letter", str("Cover letter.")), prop("proposed_amount", num("Proposed amount.")), prop("currency", str("Currency."))),
		"ResponseView":           obj(req("id", uuidStr()), req("order_id", uuidStr()), prop("executor_id", uuidStr()), prop("cover_letter", str("Cover letter.")), prop("proposed_amount", num("Proposed amount.")), req("currency", str("Currency.")), req("status", enumStr("draft", "payment_pending", "submitted", "accepted", "rejected", "cancelled")), req("is_paid", boolProp("Payment flag.")), prop("paid_at", dateTime()), req("created_at", dateTime()), req("updated_at", dateTime()), prop("order_title", str("Order title."))),
		"ResponsesListResponse":  listSchema("ResponseView"),
		"SubmitResponsePayload":  obj(req("response", ref("ResponseView")), req("payment", ref("SubmitPaymentNextStep"))),
		"Selection":              obj(req("order_id", uuidStr()), req("order_status", str("Order status.")), prop("selected_response_id", uuidStr()), prop("selected_executor_id", uuidStr()), prop("selected_response_status", str("Response status."))),
		"CreateReviewRequest":    obj(req("rating", intProp("Rating from 1 to 5.")), prop("comment", str("Review comment."))),
		"Review":                 obj(req("id", uuidStr()), req("order_id", uuidStr()), req("client_id", uuidStr()), req("executor_id", uuidStr()), req("rating", intProp("Rating from 1 to 5.")), prop("comment", str("Review comment.")), req("created_at", dateTime()), req("updated_at", dateTime())),
		"ReviewsListResponse":    listSchema("Review"),
		"RatingInfo":             obj(req("executor_id", uuidStr()), req("reviews_count_total", intProp("Total reviews.")), req("reviews_count_recent", intProp("Recent reviews counted.")), req("avg_rating_recent", num("Average rating in recent window.")), req("avg_rating_total", num("Total average rating."))),
		"Sanction":               obj(req("id", uuidStr()), req("executor_id", uuidStr()), req("status", enumStr("active", "resolved", "expired")), req("reason", str("Sanction reason.")), req("severity", intProp("Severity 1..5.")), req("started_at", dateTime()), prop("ends_at", dateTime()), prop("resolved_at", dateTime())),
		"SanctionsResponse":      obj(req("items", arr(ref("Sanction")))),
		"AdminSanctionsResponse": listSchema("Sanction"),

		"CreateCourseRequest":           obj(req("title", str("Course title.")), prop("description", str("Course description."))),
		"UpdateCourseRequest":           obj(prop("title", str("Course title.")), prop("description", str("Course description."))),
		"Course":                        obj(req("id", uuidStr()), prop("coach_id", uuidStr()), prop("created_by", uuidStr()), req("title", str("Course title.")), prop("description", str("Course description.")), req("status", enumStr("draft", "published", "archived")), req("created_at", dateTime()), req("updated_at", dateTime())),
		"CoursesListResponse":           listSchema("Course"),
		"CourseDetailResponse":          obj(req("course", ref("Course")), req("materials", arr(ref("CourseMaterial")))),
		"CreateMaterialRequest":         obj(req("title", str("Material title.")), req("type", enumStr("video", "pdf", "link", "text")), prop("url", str("URL for video/pdf/link.")), prop("content", str("Text content for text material.")), prop("position", intProp("Sort order."))),
		"UpdateMaterialRequest":         obj(prop("title", str("Material title.")), prop("type", enumStr("video", "pdf", "link", "text")), prop("url", str("URL for video/pdf/link.")), prop("content", str("Text content for text material.")), prop("position", intProp("Sort order."))),
		"CourseMaterial":                obj(req("id", uuidStr()), req("course_id", uuidStr()), req("title", str("Material title.")), req("type", enumStr("video", "pdf", "link", "text")), prop("url", str("Material URL.")), prop("content", str("Material content.")), req("position", intProp("Sort order.")), req("created_at", dateTime()), req("updated_at", dateTime())),
		"CreateAssignmentRequest":       obj(req("course_id", uuidStr()), req("executor_id", uuidStr()), req("source", enumStr("manual_admin", "sanction_low_rating_first", "sanction_low_rating_repeat")), prop("reason", str("Assignment reason.")), prop("due_at", dateTime())),
		"CourseAssignment":              obj(req("id", uuidStr()), req("course_id", uuidStr()), req("executor_id", uuidStr()), prop("sanction_id", uuidStr()), prop("assigned_by", uuidStr()), prop("reason", str("Reason.")), req("source", str("Source.")), req("status", enumStr("assigned", "in_progress", "completed", "overdue", "cancelled")), req("assigned_at", dateTime()), prop("due_at", dateTime()), prop("completed_at", dateTime()), req("created_at", dateTime()), req("updated_at", dateTime())),
		"CourseAssignmentsListResponse": listSchema("CourseAssignment"),

		"Notification":              obj(req("id", uuidStr()), req("user_id", uuidStr()), req("type", str("Notification type.")), req("channel", enumStr("in_app", "email", "sms")), req("status", enumStr("pending", "sent", "failed", "read")), req("payload", freeObj("Machine-readable payload.")), req("created_at", dateTime()), prop("sent_at", dateTime()), prop("read_at", dateTime()), prop("error_message", str("Delivery error."))),
		"NotificationsListResponse": listSchema("Notification"),
		"MarkAllReadResponse":       obj(req("updated", intProp("Number of rows updated."))),
		"ChatParticipant":           obj(req("user_id", uuidStr()), req("joined_at", dateTime()), prop("last_read_at", dateTime()), req("is_muted", boolProp("Muted flag."))),
		"ChatSummary":               obj(req("id", uuidStr()), req("order_id", uuidStr()), req("created_at", dateTime()), prop("last_message_at", dateTime()), prop("last_message", str("Last message preview.")), req("participants", arr(ref("ChatParticipant")))),
		"ChatDetail":                obj(req("id", uuidStr()), req("order_id", uuidStr()), req("created_at", dateTime()), req("participants", arr(ref("ChatParticipant")))),
		"ChatsListResponse":         listSchema("ChatSummary"),
		"Message":                   obj(req("id", uuidStr()), req("chat_id", uuidStr()), prop("sender_user_id", uuidStr()), req("sender_type", enumStr("user", "system")), req("body", str("Message body.")), req("created_at", dateTime()), prop("edited_at", dateTime()), prop("deleted_at", dateTime())),
		"MessagesListResponse":      listSchema("Message"),
		"SendMessageRequest":        obj(req("text", str("Message text."))),
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
	return map[string]any{
		"name":        name,
		"in":          in,
		"required":    required,
		"description": description,
		"schema":      map[string]any{"type": schemaType},
	}
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
