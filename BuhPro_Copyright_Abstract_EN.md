_Article 9-1 of the Law of the Republic of Kazakhstan "On Copyright and Related Rights" dated June 10, 1996 No. 6_

## **Abstract**

## **for a copyright object**

## **Web platform**

**BuhPro: a digital platform for accounting service orders, contractor selection, professional communication and specialist training**

**Object type:** computer program implemented as a client-server web platform

## **Authors**

**Sapargali Bekzat** — 87472013916

**Kuanysh Maral** — 87073271079

**Kazhiyev Shyngys** — 87782289809

## **Date of creation of the object:** 21.05.2026

## **Field of Application**

BuhPro is a digital platform designed to organize and control interaction between customers of accounting services, professional contractors, course coaches and system administrators. The program belongs to the field of web-based business service marketplaces, but its functional purpose is broader than a simple publication board. The project studies and automates the complete lifecycle of an accounting service: from the moment a customer formulates a task to the selection of a contractor, communication, completion of work, review, rating update and, when necessary, professional training of the specialist.

The subject area of the program is the accounting outsourcing market. In this field, the quality of service depends not only on the professional skills of a contractor, but also on the clarity of the task, transparency of communication, timely status control, the ability to compare responses and the existence of a reputation mechanism. In a traditional workflow, these elements are often separated. A customer may publish a request in a messenger, receive informal replies, discuss the details in private chat, transfer files through another channel and evaluate the result verbally. As a consequence, the process becomes difficult to verify, the history of decisions is fragmented, and the quality of the contractor is not connected with further professional development.

BuhPro addresses this problem by creating a single information environment where the accounting service is represented as a structured digital process. Each major action receives a formal state, each participant acts within a defined role, and the system stores the history of orders, responses, chats, payments, reviews, sanctions, notifications and training assignments. Therefore, the program can be considered not only as a practical platform for placing accounting tasks, but also as a software model for researching the digital organization of trust in professional services.

The object of research implemented in the program is the interaction process between a customer and an accounting contractor. The subject of research is the set of software mechanisms that make this interaction controllable: role-based access, order lifecycle management, response lifecycle management, payment state simulation, contractor selection, in-system communication, rating recalculation, sanctions and educational follow-up. The practical goal of the project is to reduce uncertainty in the accounting service market and provide participants with a predictable digital workflow.

From the customer's point of view, the central function of the program is the creation of an order. The customer describes the task, selects a service category, specifies the budget, region and expected deadline, and receives a calculated placement cost. This interface forces the task to be formulated in a structured way. As a result, the contractor receives not a vague message, but a formal description of a business problem, such as bookkeeping for a limited liability partnership, tax optimization for an individual entrepreneur, preparation of financial statements or an internal audit.

**==> picture [client_create_order] intentionally omitted <==**

Figure 1. Customer order creation and placement cost calculation

The creation form is important because it converts a business need into data that can be processed by the system. The title gives a short semantic label to the request. The category allows filtering and later statistical grouping. The description provides contextual details for the contractor. The budget defines the economic boundary of the task. The deadline indicates the expected time constraint, and the region clarifies whether the service can be performed remotely or requires local knowledge. These fields support both practical use and analytical interpretation of the platform's activity.

After the order is created, it appears in the customer's dashboard. The dashboard separates active orders, orders in progress and completed orders. It also distinguishes drafts from published orders, which is essential for preserving the natural sequence of work. The customer can first create a draft, then clarify its content, and only after that send it for publication. This reduces the risk of incomplete or inaccurate accounting requests being immediately exposed to contractors.

**==> picture [client_dashboard] intentionally omitted <==**

Figure 2. Customer dashboard with the list of orders

The dashboard also performs a monitoring function. It gives the customer a compact view of the current state of cooperation with contractors. Instead of searching through separate chats or documents, the user sees all relevant orders in one interface. This is particularly significant for accounting services, because the same customer may simultaneously need tax consultation, financial reporting, bookkeeping support and audit-related assistance. The platform therefore supports parallel management of several service requests.

Order editing is implemented through a modal window. This design decision has practical meaning: the customer can correct the title, description and budget while remaining within the order list context. The program therefore supports an iterative approach to task formulation. In real accounting practice, requirements often become clearer only after the customer starts writing them down. The ability to revise the order before final submission makes the platform closer to the real decision-making process.

**==> picture [client_edit_order_modal] intentionally omitted <==**

Figure 3. Editing order parameters in the customer dashboard

From the contractor's point of view, the platform studies a different part of the workflow: search, filtering and decision-making before submitting a response. Contractors can view published orders, filter them by category, city and keywords, and sort them by recency. Each order card contains the budget, publication date, category and short description. This enables the contractor to estimate whether the task is economically attractive and whether it matches their competence.

**==> picture [executor_orders_search] intentionally omitted <==**

Figure 4. Contractor search and filtering of published orders

The search interface is a key element of the platform because it transforms a general marketplace into a targeted professional environment. A tax consultant can focus on tax-related requests, a bookkeeping specialist can search for monthly accounting tasks, and an auditor can select audit or reporting assignments. In this way, the program supports not only the availability of orders, but also the matching between the content of the accounting task and the contractor's professional profile.

The contractor dashboard complements order search with personal performance analytics. It displays the contractor's rating, number of responses, active work, completed tasks, profile information and productivity indicators. This part of the system performs a reputation-forming function. The contractor does not simply send proposals; their actions become measurable indicators that influence trust. For the customer, rating and reviews become an additional basis for choosing a specialist. For the contractor, the dashboard becomes a feedback mechanism showing how professional activity is reflected in the system.

**==> picture [executor_dashboard] intentionally omitted <==**

Figure 5. Contractor dashboard with orders, rating and productivity indicators

The role of ratings in BuhPro is not limited to displaying a number. The platform connects ratings with the broader quality control process. After an order is completed, the customer can leave a review. The review affects the contractor's profile, and low ratings may generate sanctions or lead to educational assignments. This design is important because it creates a closed quality loop. The system does not only record poor performance; it can also initiate a corrective training path.

Communication between participants is implemented in a separate chat module. The chat is linked to a particular order, which distinguishes it from an ordinary messenger. The discussion remains connected to the business context, and each conversation belongs to a specific task. In the current implementation, the chat module is REST-based and includes message history, read-state management and participant access control. The result is a verifiable communication channel between the customer and the contractor.

**==> picture [chat] intentionally omitted <==**

Figure 6. Chat interface between order participants

The chat module is especially relevant for accounting services because task details often require clarification. A contractor may need additional documents, explanations about business turnover, tax regime, previous reporting periods or deadlines. When such communication is stored in the same platform as the order, the interaction becomes easier to track and audit. This reduces the risk of losing important information and improves accountability between the parties.

Another important direction of the project is training and competence development. The platform includes the role of a coach who manages educational courses. The coach dashboard displays the number of courses, the number of published courses, the completion rate, drafts and popular courses. This component links the marketplace logic with an educational subsystem. The platform therefore does not only distribute accounting tasks, but also supports the growth of professional competencies.

**==> picture [coach_dashboard] intentionally omitted <==**

Figure 7. Coach dashboard for managing educational courses

The creation of a course is implemented as a multi-step process: basic information, content, price and access settings, and preview. This structure is important because an educational product must be described, organized and presented before it becomes useful to students. The preview shows how the course will look to learners, including its price, duration, certificate availability and tags. For BuhPro, courses are not an unrelated feature; they are part of the platform's quality improvement mechanism.

**==> picture [course_create_preview] intentionally omitted <==**

Figure 8. Preview of a created educational course

The internal logic of BuhPro is built around several interconnected entities. An order may pass through the states of draft, payment pending, published, in progress, completed or cancelled. A contractor response also has its own lifecycle: draft, payment pending, submitted, accepted, rejected or cancelled. Payment transactions record the economic stage of publishing an order or submitting a response. When the customer selects a contractor, the selected response is accepted, other responses are rejected, and the order moves into progress. After completion, the customer can leave a review, and the review influences the contractor's rating.

This lifecycle approach is essential for the research value of the program. It allows each business event to be represented as a transition between states rather than as an informal action. For example, publishing an order is not simply the display of a card on a page; it is the result of validation, payment status handling and transition from preparation to public availability. Similarly, choosing a contractor is not only a button click; it changes the state of the order, the state of responses and the communication context. Such modeling makes the platform predictable and suitable for further development.

The program also includes a notification subsystem. Notifications are generated when important events occur: order publication, response submission, contractor selection, order completion, review creation, sanction creation, course assignment and chat message receipt. Notifications reduce the need for users to manually check every section of the platform. From a system perspective, they also help connect independent modules into one coherent workflow. An event in the order module may create an update in the notification module, a message in the chat module, or a course assignment in the training module.

The administrative role provides supervision over the platform. Administrators can inspect orders, sanctions, course assignments, notifications, chats and other platform data. In a production environment, such functions are necessary for moderation, dispute analysis and operational control. In the current implementation, the administrative module also supports demonstration and development scenarios, including payment confirmation endpoints for local testing. This makes it possible to test the full order and response lifecycle without integrating a real payment provider at the early stage of development.

Security and access control are implemented through authentication, role-based authorization and scoped endpoints. Customers can create and manage their own orders. Contractors can submit responses and manage their own responses. Coaches can create and manage courses. Administrators have broader read and control capabilities. The program uses JWT-based authentication and separates public endpoints from protected ones. This is important because accounting tasks may contain financial and business-sensitive information, and each participant must see only the data relevant to their role.

The backend architecture follows a layered structure. HTTP handlers receive and validate requests, services contain business rules and access checks, repositories communicate with the PostgreSQL database, and platform-level modules provide infrastructure functions such as logging, metrics, payments, storage and database migrations. This separation makes the program more maintainable. It also supports the research purpose of the project: individual business processes can be studied, tested and improved without mixing interface logic, business rules and SQL code in one place.

The frontend architecture is responsible for presenting the same business logic to different user roles. The user interface is role-aware: a customer sees order creation and order management; a contractor sees search, responses and personal performance; a coach sees course creation and analytics; an administrator sees control and supervision tools. This role-sensitive interface is important because the same platform must support different goals without overloading each user with irrelevant functions.

The practical significance of the program lies in the creation of a managed digital environment for accounting services. For customers, BuhPro reduces uncertainty when searching for a specialist, because the order is structured, responses are collected centrally, and communication is attached to the order. For contractors, the system provides a stable channel for finding work, building reputation and receiving feedback. For coaches, it provides a tool for publishing and managing educational content. For administrators, it provides observability and control over platform processes.

The economic significance of the platform is related to the reduction of transaction costs. In the absence of a specialized platform, customers spend time searching for specialists, comparing offers manually and maintaining communication through external channels. Contractors spend time looking for suitable tasks and proving their competence repeatedly. BuhPro reduces these costs by standardizing the order format, providing search and filters, keeping histories of responses and reviews, and making reputation visible within the platform.

The social significance of the platform is connected with the professionalization of digital accounting services. Small businesses and individual entrepreneurs often require accounting support but do not have internal accounting departments. A digital platform can make professional services more accessible while giving specialists an opportunity to offer their work in a transparent environment. The integration of training courses further increases the value of the system because it supports continuous improvement rather than only one-time service exchange.

At the current stage, the system focuses on the core MVP processes: authentication, profiles, orders, responses, selection, reviews, ratings, sanctions, courses, notifications, chats, uploads and demonstration payment flows. Some advanced production features may be developed later, such as real payment callbacks, webhooks, real-time chat through WebSocket, advanced moderation, detailed dispute resolution, email or SMS delivery and extended course analytics. These limitations do not reduce the value of the current program; rather, they define the boundary of the implemented research prototype.

## **Main Technical Characteristics, Programming Language and Type of Implementing Computer**

**Hardware and software requirements:**

The client side of the program is launched in a modern web browser and does not require installation of a separate desktop application. A user needs a personal computer or laptop with Internet access, an Intel Core i3 class processor or higher, at least 4 GB of RAM, a monitor with a resolution of at least 1024x768 and a current version of Google Chrome, Microsoft Edge, Mozilla Firefox or Safari. Since the platform is web-based, the main requirement for the end user is stable access to the server through a browser.

The server side of the program can run on a computer, local development machine or virtual server with Windows, Linux or macOS. For local deployment, Docker and Docker Compose are used together with PostgreSQL, a Go backend service and a Next.js frontend application. Recommended server characteristics include an x86-64 processor, at least 4 GB of RAM and at least 1 GB of free disk space for the application, migrations, local file storage and database data. For larger deployments, memory, CPU and database storage should be scaled according to the number of active users, orders, files and chat messages.

**Program architecture:**

BuhPro is implemented as a web platform with separated client and server parts. The frontend provides user interfaces for customers, contractors, coaches and administrators. The backend exposes a REST API, validates access rights, stores business entities, controls order and response states, manages authentication, notifications, chats, courses, ratings, sanctions and file uploads.

The backend is built according to a modular layered architecture. The main modules include authentication, profiles, orders, responses, contractor selection, reviews, ratings and sanctions, courses and course assignments, notifications, chats, uploads, attachments, wallets and development payment operations. Data is stored in PostgreSQL, and the database structure is maintained through migration files. The system uses JSON API responses, pagination, request identifiers, logging, recovery middleware and Prometheus-compatible metrics.

The program supports the following user roles:

- Customer: creates accounting service orders, manages drafts, publishes orders, reviews responses, selects a contractor, completes work and leaves reviews.
- Contractor: searches for published orders, submits responses, communicates with customers, performs work, receives ratings and monitors sanctions or course assignments.
- Coach: creates and manages educational courses, publishes course content and monitors course-related indicators.
- Administrator: supervises platform activity, reviews system data, manages sanctions, notifications, course assignments and development payment scenarios.

The main data flows of the platform include:

- Order flow: draft creation, editing, submission, payment pending state, publication, contractor selection, work in progress, completion or cancellation.
- Response flow: contractor draft, submission, payment pending state, submitted response, acceptance, rejection or cancellation.
- Communication flow: chat creation, message sending, message reading and notification of the second participant.
- Quality flow: order completion, review creation, rating update, sanction creation and possible course assignment.
- Training flow: course creation, publication, assignment to a contractor and completion tracking.

This architecture makes the program extensible. New payment providers, additional notification channels, real-time messaging, advanced course features or moderation tools can be added without changing the entire system structure.

**Installation, launch and removal of the program:**

To launch the backend locally, environment variables must be prepared, PostgreSQL must be started and database migrations must be applied:

```powershell
Copy-Item .env.example .env
docker compose up -d postgres
make migrate-up
make run
```

The frontend is launched separately in the client application directory:

```powershell
pnpm install
pnpm dev
```

After launch, the user opens the platform in a browser at:

```text
http://localhost:3000
```

The backend API is available by default at:

```text
http://localhost:8080
```

When using Docker Compose, the backend service, PostgreSQL database and additional infrastructure can be launched with one command:

```powershell
docker compose up --build -d
```

To stop the local version of the program, the following command is used:

```powershell
docker compose down
```

If persistent local database data also needs to be removed, Docker volumes can be deleted intentionally by the developer or administrator after backup and verification.

## **Programming Language**

This software product was created using the following technologies.

Backend: Go 1.23, Gin web framework, pgx for PostgreSQL access, JWT for authentication, PostgreSQL migrations, structured logging, request middleware, Prometheus-compatible metrics and Docker-based deployment.

Frontend: TypeScript, React, Next.js, Tailwind CSS, Radix UI, lucide-react, SWR and Firebase Web SDK for client-side integration scenarios.

Database and infrastructure: PostgreSQL, Docker, Docker Compose, local file storage for uploaded materials and REST API as the main communication mechanism between the frontend and backend.

The selected technology stack is suitable for the project because Go provides high performance and simple deployment for backend services, PostgreSQL provides reliable relational storage for orders, responses, users and chats, and Next.js provides a flexible web interface for role-based user scenarios. Together, these technologies form a complete web platform that can be launched locally for demonstration and extended for production deployment.
