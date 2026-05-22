param(
    [string[]]$BaseUrls = @("http://localhost:8080"),
    [string]$Password = "DemoPass123",
    [switch]$ReadOnly,
    [int]$TimeoutSec = 20
)

$ErrorActionPreference = "Stop"

$DemoUsers = @{
    client   = "demo.client@buhpro.local"
    executor = "demo.executor@buhpro.local"
    coach    = "demo.coach@buhpro.local"
    admin    = "demo.admin@buhpro.local"
}

$script:Passed = 0
$script:Failed = 0

function Normalize-BaseUrl {
    param([string]$Url)
    $value = $Url.Trim()
    if ($value.EndsWith("/")) {
        return $value.TrimEnd("/")
    }
    return $value
}

function Convert-JsonSafe {
    param([string]$Text)
    if ([string]::IsNullOrWhiteSpace($Text)) {
        return $null
    }
    try {
        return $Text | ConvertFrom-Json
    } catch {
        return $Text
    }
}

function Read-ErrorBody {
    param($ErrorRecord)
    if ($null -ne $ErrorRecord.ErrorDetails -and -not [string]::IsNullOrWhiteSpace($ErrorRecord.ErrorDetails.Message)) {
        return [string]$ErrorRecord.ErrorDetails.Message
    }
    if ($null -eq $ErrorRecord.Exception.Response) {
        return ""
    }
    try {
        $stream = $ErrorRecord.Exception.Response.GetResponseStream()
        if ($null -eq $stream) {
            return ""
        }
        $reader = New-Object System.IO.StreamReader($stream)
        return $reader.ReadToEnd()
    } catch {
        return ""
    }
}

function Write-Check {
    param(
        [string]$Name,
        [bool]$Ok,
        [string]$Details = ""
    )

    if ($Ok) {
        $script:Passed++
        Write-Host ("[OK]   {0} {1}" -f $Name, $Details) -ForegroundColor Green
    } else {
        $script:Failed++
        Write-Host ("[FAIL] {0} {1}" -f $Name, $Details) -ForegroundColor Red
    }
}

function Format-HttpDetail {
    param(
        [int]$Status,
        [string]$Content,
        [bool]$Ok
    )
    $detail = "HTTP $Status"
    if (-not $Ok -and -not [string]::IsNullOrWhiteSpace($Content)) {
        $short = $Content.Replace("`r", " ").Replace("`n", " ")
        if ($short.Length -gt 280) {
            $short = $short.Substring(0, 280) + "..."
        }
        $detail = "$detail $short"
    }
    return $detail
}

function Invoke-Api {
    param(
        [string]$BaseUrl,
        [string]$Method,
        [string]$Path,
        [object]$Body = $null,
        [string]$Token = "",
        [int[]]$Expected = @(200),
        [string]$Name
    )

    $headers = @{}
    if (-not [string]::IsNullOrWhiteSpace($Token)) {
        $headers["Authorization"] = "Bearer $Token"
    }

    $params = @{
        Uri             = "$BaseUrl$Path"
        Method          = $Method
        Headers         = $headers
        TimeoutSec      = $TimeoutSec
        UseBasicParsing = $true
        ErrorAction     = "Stop"
    }

    if ($null -ne $Body) {
        $params["ContentType"] = "application/json"
        $params["Body"] = ($Body | ConvertTo-Json -Depth 30 -Compress)
    }

    $status = 0
    $content = ""
    $json = $null

    try {
        $resp = Invoke-WebRequest @params
        $status = [int]$resp.StatusCode
        $content = [string]$resp.Content
        $json = Convert-JsonSafe $content
    } catch {
        if ($_.Exception.Response) {
            $status = [int]$_.Exception.Response.StatusCode
            $content = Read-ErrorBody $_
            $json = Convert-JsonSafe $content
        } else {
            $content = $_.Exception.Message
        }
    }

    $ok = $Expected -contains $status
    Write-Check $Name $ok (Format-HttpDetail -Status $status -Content $content -Ok $ok)

    return [pscustomobject]@{
        Ok      = $ok
        Status  = $status
        Json    = $json
        Content = $content
    }
}

function Invoke-MultipartApi {
    param(
        [string]$BaseUrl,
        [string]$Method,
        [string]$Path,
        [hashtable]$Fields = @{},
        [object[]]$Files = @(),
        [string]$Token = "",
        [int[]]$Expected = @(200),
        [string]$Name
    )

    Add-Type -AssemblyName System.Net.Http

    $client = [System.Net.Http.HttpClient]::new()
    $client.Timeout = [TimeSpan]::FromSeconds($TimeoutSec)
    if (-not [string]::IsNullOrWhiteSpace($Token)) {
        $client.DefaultRequestHeaders.Authorization = [System.Net.Http.Headers.AuthenticationHeaderValue]::new("Bearer", $Token)
    }

    $form = [System.Net.Http.MultipartFormDataContent]::new()
    $request = $null
    $status = 0
    $content = ""
    $json = $null

    try {
        foreach ($key in $Fields.Keys) {
            $value = $Fields[$key]
            if ($null -eq $value) {
                continue
            }
            if (($value -is [System.Array]) -and -not ($value -is [byte[]])) {
                foreach ($item in $value) {
                    $form.Add([System.Net.Http.StringContent]::new([string]$item, [System.Text.Encoding]::UTF8), $key)
                }
            } else {
                $form.Add([System.Net.Http.StringContent]::new([string]$value, [System.Text.Encoding]::UTF8), $key)
            }
        }

        foreach ($file in $Files) {
            $bytes = [byte[]]$file.Bytes
            $fileContent = [System.Net.Http.ByteArrayContent]::new($bytes)
            $fileContent.Headers.ContentType = [System.Net.Http.Headers.MediaTypeHeaderValue]::Parse([string]$file.ContentType)
            $form.Add($fileContent, [string]$file.FieldName, [string]$file.FileName)
        }

        $request = [System.Net.Http.HttpRequestMessage]::new([System.Net.Http.HttpMethod]::new($Method), "$BaseUrl$Path")
        $request.Content = $form
        $resp = $client.SendAsync($request).GetAwaiter().GetResult()
        $status = [int]$resp.StatusCode
        $content = $resp.Content.ReadAsStringAsync().GetAwaiter().GetResult()
        $json = Convert-JsonSafe $content
    } catch {
        $content = $_.Exception.Message
    } finally {
        if ($null -ne $request) { $request.Dispose() }
        $form.Dispose()
        $client.Dispose()
    }

    $ok = $Expected -contains $status
    Write-Check $Name $ok (Format-HttpDetail -Status $status -Content $content -Ok $ok)

    return [pscustomobject]@{
        Ok      = $ok
        Status  = $status
        Json    = $json
        Content = $content
    }
}

function Require-Value {
    param(
        [object]$Value,
        [string]$Name
    )
    $ok = $null -ne $Value -and -not [string]::IsNullOrWhiteSpace([string]$Value)
    Write-Check $Name $ok
    return $ok
}

function Get-Items {
    param([object]$ListResult)
    if ($null -eq $ListResult -or $null -eq $ListResult.items) {
        return @()
    }
    return @($ListResult.items)
}

function Get-FirstItemId {
    param([object]$ListResult)
    $items = @(Get-Items $ListResult)
    if ($items.Count -lt 1) {
        return ""
    }
    return $items[0].id
}

function Get-FirstItemWhere {
    param(
        [object]$ListResult,
        [string]$Property,
        [string]$Value
    )
    foreach ($item in @(Get-Items $ListResult)) {
        if ([string]$item.$Property -eq $Value) {
            return $item
        }
    }
    return $null
}

function Login-DemoUser {
    param(
        [string]$BaseUrl,
        [string]$Role,
        [string]$Email
    )

    $res = Invoke-Api `
        -BaseUrl $BaseUrl `
        -Method "POST" `
        -Path "/api/v1/auth/login" `
        -Body @{ email = $Email; password = $Password } `
        -Expected @(200) `
        -Name "login $Role <$Email>"

    if (-not $res.Ok) {
        return $null
    }

    if (-not (Require-Value $res.Json.access_token "token exists for $Role")) {
        return $null
    }

    return [pscustomobject]@{
        Role         = $Role
        Email        = $Email
        UserId       = $res.Json.user_id
        AccessToken  = $res.Json.access_token
        RefreshToken = $res.Json.refresh_token
    }
}

function Get-TestPdfBytes {
    return [System.Text.Encoding]::ASCII.GetBytes("%PDF-1.4`n1 0 obj <<>> endobj`ntrailer <<>>`n%%EOF")
}

function Get-TestPngBytes {
    return [Convert]::FromBase64String("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=")
}

function New-TestFile {
    param(
        [string]$FieldName,
        [string]$FileName,
        [string]$ContentType,
        [byte[]]$Bytes
    )
    return [pscustomobject]@{
        FieldName   = $FieldName
        FileName    = $FileName
        ContentType = $ContentType
        Bytes       = $Bytes
    }
}

function Upload-TestFile {
    param(
        [string]$BaseUrl,
        [string]$Token,
        [string]$Name,
        [object]$File
    )

    $upload = Invoke-MultipartApi `
        -BaseUrl $BaseUrl `
        -Method "POST" `
        -Path "/api/v1/files" `
        -Token $Token `
        -Files @($File) `
        -Expected @(201) `
        -Name $Name

    if (-not $upload.Ok) {
        return $null
    }
    $id = Get-FirstItemId $upload.Json
    if (-not (Require-Value $id "$Name id")) {
        return $null
    }
    return $id
}

function Submit-ExecutorLead {
    param(
        [string]$BaseUrl,
        [string]$Email,
        [string]$Name
    )

    $pdf = Get-TestPdfBytes
    return Invoke-MultipartApi `
        -BaseUrl $BaseUrl `
        -Method "POST" `
        -Path "/api/v1/leads/executor" `
        -Fields @{
            email            = $Email
            password         = $Password
            first_name       = "Lead"
            last_name        = "Tester"
            iin              = "123456789012"
            phone            = "+77070000000"
            city             = "Almaty"
            experience_level = "senior"
            specializations  = '["tax","audit"]'
            education        = "Higher finance education"
            work_format      = "remote"
            about            = "Executor lead created by the full API test."
            terms_accepted   = "true"
            source           = "full_api_test"
        } `
        -Files @(
            (New-TestFile -FieldName "identity_document" -FileName "identity.pdf" -ContentType "application/pdf" -Bytes $pdf),
            (New-TestFile -FieldName "education_document" -FileName "education.pdf" -ContentType "application/pdf" -Bytes $pdf)
        ) `
        -Expected @(201) `
        -Name $Name
}

function New-OrderDraft {
    param(
        [string]$BaseUrl,
        [string]$Token,
        [string]$Title,
        [double]$Budget = 12000
    )
    return Invoke-Api `
        -BaseUrl $BaseUrl `
        -Method "POST" `
        -Path "/api/v1/orders" `
        -Token $Token `
        -Body @{
            title         = $Title
            description   = "Full API test bookkeeping order created from PowerShell."
            category_slug = "tax"
            budget_amount = $Budget
            currency      = "KZT"
            region        = "online"
        } `
        -Expected @(201) `
        -Name "orders POST $Title"
}

function New-PublishedOrder {
    param(
        [string]$BaseUrl,
        [object]$Client,
        [string]$Title,
        [double]$Budget = 12000
    )
    $draft = New-OrderDraft -BaseUrl $BaseUrl -Token $Client.AccessToken -Title $Title -Budget $Budget
    if (-not $draft.Ok -or -not (Require-Value $draft.Json.id "$Title id")) {
        return $null
    }
    $orderId = $draft.Json.id
    $submit = Invoke-Api `
        -BaseUrl $BaseUrl `
        -Method "POST" `
        -Path "/api/v1/orders/my/$orderId/submit" `
        -Token $Client.AccessToken `
        -Expected @(200) `
        -Name "orders/my/{id}/submit POST $Title"
    if (-not $submit.Ok) {
        return $null
    }
    Write-Check "$Title is published" ($submit.Json.order.status -eq "published") "status=$($submit.Json.order.status)"
    return $orderId
}

function Run-FullApiTestsForBaseUrl {
    param([string]$InputBaseUrl)

    $baseUrl = Normalize-BaseUrl $InputBaseUrl
    $stamp = Get-Date -Format "yyyyMMddHHmmss"
    Write-Host ""
    Write-Host "=== BuhPro FULL API TEST: $baseUrl ===" -ForegroundColor Cyan

    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/healthz" -Expected @(200) -Name "healthz" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/readyz" -Expected @(200) -Name "readyz" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/ping" -Expected @(200) -Name "ping" | Out-Null

    $tempEmail = "test.user_$stamp@buhpro.local"
    Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/auth/register" -Body @{
        email        = $tempEmail
        password     = $Password
        role         = "client"
        profile_name = "Test User"
    } -Expected @(201) -Name "auth/register new user" | Out-Null

    $client = Login-DemoUser -BaseUrl $baseUrl -Role "client" -Email $DemoUsers.client
    $executor = Login-DemoUser -BaseUrl $baseUrl -Role "executor" -Email $DemoUsers.executor
    $coach = Login-DemoUser -BaseUrl $baseUrl -Role "coach" -Email $DemoUsers.coach
    $admin = Login-DemoUser -BaseUrl $baseUrl -Role "admin" -Email $DemoUsers.admin

    if ($null -eq $client -or $null -eq $executor -or $null -eq $coach -or $null -eq $admin) {
        Write-Host "Skipping deeper checks for $baseUrl because core login failed." -ForegroundColor Yellow
        return
    }

    $tempLogin = Login-DemoUser -BaseUrl $baseUrl -Role "temp_client" -Email $tempEmail
    if ($null -ne $tempLogin) {
        Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/auth/refresh" -Body @{ refresh_token = $tempLogin.RefreshToken } -Expected @(200) -Name "auth/refresh" | Out-Null
        Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/auth/logout" -Token $tempLogin.AccessToken -Body @{ refresh_token = $tempLogin.RefreshToken } -Expected @(200) -Name "auth/logout" | Out-Null
    }

    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/auth/me" -Token $client.AccessToken -Expected @(200) -Name "auth/me client" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/auth/me" -Token $executor.AccessToken -Expected @(200) -Name "auth/me executor" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/profile" -Token $client.AccessToken -Expected @(200) -Name "profile GET" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "PATCH" -Path "/api/v1/profile" -Token $client.AccessToken -Body @{ about = "Full API test profile update"; website = "https://client-demo.buhpro.local" } -Expected @(200) -Name "profile PATCH" | Out-Null

    $avatarUploadId = Upload-TestFile `
        -BaseUrl $baseUrl `
        -Token $client.AccessToken `
        -Name "files POST avatar" `
        -File (New-TestFile -FieldName "file" -FileName "avatar.png" -ContentType "image/png" -Bytes (Get-TestPngBytes))
    if ($null -ne $avatarUploadId) {
        Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/files/$avatarUploadId" -Token $client.AccessToken -Expected @(200) -Name "files/{id} GET avatar" | Out-Null
        Invoke-Api -BaseUrl $baseUrl -Method "PATCH" -Path "/api/v1/profile/avatar" -Token $client.AccessToken -Body @{ upload_id = $avatarUploadId } -Expected @(200) -Name "profile/avatar PATCH" | Out-Null
        Invoke-Api -BaseUrl $baseUrl -Method "DELETE" -Path "/api/v1/profile/avatar" -Token $client.AccessToken -Expected @(200) -Name "profile/avatar DELETE" | Out-Null
    }

    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/my/wallet" -Token $client.AccessToken -Expected @(200) -Name "my/wallet" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/admin/wallets/$($client.UserId)" -Token $admin.AccessToken -Expected @(200) -Name "admin/wallets/{userId} GET" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/admin/wallets/$($client.UserId)/credit" -Token $admin.AccessToken -Body @{ amount = 1000000; reason = "full_api_test_credit" } -Expected @(200) -Name "admin/wallets/{userId}/credit" | Out-Null

    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/my/files" -Token $client.AccessToken -Expected @(200) -Name "my/files GET" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/attachments?target_type=order_attachment&target_id=00000000-0000-0000-0000-000000000000" -Expected @(200) -Name "attachments GET empty target" | Out-Null

    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/reviews?target_type=user&target_id=$($executor.UserId)" -Expected @(200) -Name "reviews GET list by target" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/ratings?target_type=user&target_id=$($executor.UserId)" -Expected @(200) -Name "ratings GET summary" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/executors/$($executor.UserId)/reviews" -Expected @(200) -Name "executors/{id}/reviews GET" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/executors/$($executor.UserId)/rating" -Expected @(200) -Name "executors/{id}/rating GET" | Out-Null

    if (-not $ReadOnly) {
        $leadReject = Submit-ExecutorLead -BaseUrl $baseUrl -Email "lead.reject_$stamp@buhpro.local" -Name "leads/executor POST reject flow"
        if ($leadReject.Ok -and (Require-Value $leadReject.Json.lead_id "lead reject id")) {
            $leadId = $leadReject.Json.lead_id
            Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/admin/executor-leads" -Token $admin.AccessToken -Expected @(200) -Name "admin/executor-leads GET" | Out-Null
            Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/admin/executor-leads/$leadId" -Token $admin.AccessToken -Expected @(200) -Name "admin/executor-leads/{id} GET" | Out-Null
            Invoke-Api -BaseUrl $baseUrl -Method "PATCH" -Path "/api/v1/admin/executor-leads/$leadId/status" -Token $admin.AccessToken -Body @{ status = "in_review"; notes = "Full API test review" } -Expected @(200) -Name "admin/executor-leads/{id}/status PATCH" | Out-Null
            Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/admin/executor-leads/$leadId/reject" -Token $admin.AccessToken -Body @{ reason = "Full API test rejection" } -Expected @(200) -Name "admin/executor-leads/{id}/reject POST" | Out-Null
        }

        $leadApprove = Submit-ExecutorLead -BaseUrl $baseUrl -Email "lead.approve_$stamp@buhpro.local" -Name "leads/executor POST approve flow"
        if ($leadApprove.Ok -and (Require-Value $leadApprove.Json.lead_id "lead approve id")) {
            Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/admin/executor-leads/$($leadApprove.Json.lead_id)/approve" -Token $admin.AccessToken -Body @{ notes = "Full API test approval" } -Expected @(201) -Name "admin/executor-leads/{id}/approve POST" | Out-Null
        }
    }

    $courseCreate = Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/coach/courses" -Token $coach.AccessToken -Body @{
        title       = "Full API Course $stamp"
        description = "Full API test course."
    } -Expected @(201) -Name "coach/courses POST"

    if ($courseCreate.Ok -and (Require-Value $courseCreate.Json.id "created course id")) {
        $courseId = $courseCreate.Json.id
        Invoke-Api -BaseUrl $baseUrl -Method "PATCH" -Path "/api/v1/coach/courses/$courseId" -Token $coach.AccessToken -Body @{ title = "Full API Course Updated $stamp" } -Expected @(200) -Name "coach/courses/{id} PATCH" | Out-Null
        Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/coach/courses/$courseId" -Token $coach.AccessToken -Expected @(200) -Name "coach/courses/{id} GET" | Out-Null
        Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/coach/courses" -Token $coach.AccessToken -Expected @(200) -Name "coach/courses GET list" | Out-Null

        $materialCreate = Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/coach/courses/$courseId/materials" -Token $coach.AccessToken -Body @{
            title    = "Text Material"
            type     = "text"
            content  = "Full API test material content."
            position = 1
        } -Expected @(201) -Name "coach/courses/{id}/materials POST"
        if ($materialCreate.Ok -and (Require-Value $materialCreate.Json.id "created material id")) {
            $materialId = $materialCreate.Json.id
            Invoke-Api -BaseUrl $baseUrl -Method "PATCH" -Path "/api/v1/coach/courses/$courseId/materials/$materialId" -Token $coach.AccessToken -Body @{ title = "Text Material Updated"; content = "Updated content." } -Expected @(200) -Name "coach/courses/{id}/materials/{mId} PATCH" | Out-Null
            Invoke-Api -BaseUrl $baseUrl -Method "DELETE" -Path "/api/v1/coach/courses/$courseId/materials/$materialId" -Token $coach.AccessToken -Expected @(200) -Name "coach/courses/{id}/materials/{mId} DELETE" | Out-Null
        }

        Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/coach/courses/$courseId/publish" -Token $coach.AccessToken -Expected @(200) -Name "coach/courses/{id}/publish POST" | Out-Null
        Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/courses" -Token $executor.AccessToken -Expected @(200) -Name "courses GET list" | Out-Null
        Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/courses/$courseId" -Token $executor.AccessToken -Expected @(200) -Name "courses/{id} GET" | Out-Null

        $assignCreate = Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/admin/course-assignments" -Token $admin.AccessToken -Body @{
            executor_id = $executor.UserId
            course_id   = $courseId
            source      = "manual_admin"
            reason      = "Full API test assignment"
        } -Expected @(201) -Name "admin/course-assignments POST"
        Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/admin/course-assignments" -Token $admin.AccessToken -Expected @(200) -Name "admin/course-assignments GET" | Out-Null

        if ($assignCreate.Ok -and (Require-Value $assignCreate.Json.id "created assignment id")) {
            $assignId = $assignCreate.Json.id
            Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/my/course-assignments" -Token $executor.AccessToken -Expected @(200) -Name "my/course-assignments GET" | Out-Null
            Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/my/course-assignments/$assignId" -Token $executor.AccessToken -Expected @(200) -Name "my/course-assignments/{id} GET" | Out-Null
            Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/my/course-assignments/$assignId/mark-completed" -Token $executor.AccessToken -Expected @(200) -Name "my/course-assignments/{id}/mark-completed POST" | Out-Null
        }
        Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/coach/courses/$courseId/archive" -Token $coach.AccessToken -Expected @(200) -Name "coach/courses/{id}/archive POST" | Out-Null
    }

    $directChat = Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/my/chats" -Token $client.AccessToken -Body @{ participant_id = $executor.UserId } -Expected @(201) -Name "my/chats POST direct"
    if ($directChat.Ok -and (Require-Value $directChat.Json.chat_id "direct chat id")) {
        $directChatId = $directChat.Json.chat_id
        Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/my/chats/$directChatId" -Token $client.AccessToken -Expected @(200) -Name "my/chats/{id} GET direct" | Out-Null
    }

    if ($ReadOnly) {
        Write-Host "ReadOnly mode: skipped create/submit/payment/selection lifecycle." -ForegroundColor Yellow
        return
    }

    $mainDraft = New-OrderDraft -BaseUrl $baseUrl -Token $client.AccessToken -Title "Full API Main Order $stamp" -Budget 12000
    if (-not $mainDraft.Ok -or -not (Require-Value $mainDraft.Json.id "main order id")) {
        return
    }
    $orderId = $mainDraft.Json.id

    $attachmentUploadId = Upload-TestFile `
        -BaseUrl $baseUrl `
        -Token $client.AccessToken `
        -Name "files POST order attachment" `
        -File (New-TestFile -FieldName "file" -FileName "order-note.txt" -ContentType "text/plain" -Bytes ([System.Text.Encoding]::UTF8.GetBytes("Full API order attachment $stamp")))
    if ($null -ne $attachmentUploadId) {
        $attach = Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/attachments" -Token $client.AccessToken -Body @{
            upload_ids  = @($attachmentUploadId)
            target_type = "order_attachment"
            target_id   = $orderId
            metadata    = @{ source = "full_api_test" }
        } -Expected @(201) -Name "attachments POST order"
        Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/attachments?target_type=order_attachment&target_id=$orderId" -Expected @(200) -Name "attachments GET order" | Out-Null
        if ($attach.Ok) {
            $attachmentId = Get-FirstItemId $attach.Json
            if (Require-Value $attachmentId "created attachment id") {
                Invoke-Api -BaseUrl $baseUrl -Method "PATCH" -Path "/api/v1/attachments/reorder" -Token $client.AccessToken -Body @{ ids = @($attachmentId) } -Expected @(200) -Name "attachments PATCH reorder" | Out-Null
                Invoke-Api -BaseUrl $baseUrl -Method "DELETE" -Path "/api/v1/attachments/$attachmentId" -Token $client.AccessToken -Expected @(200) -Name "attachments DELETE" | Out-Null
            }
        }
    }

    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/orders/my" -Token $client.AccessToken -Expected @(200) -Name "orders/my GET" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/orders/my/$orderId" -Token $client.AccessToken -Expected @(200) -Name "orders/my/{id} GET" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "PATCH" -Path "/api/v1/orders/my/$orderId" -Token $client.AccessToken -Body @{ budget_amount = 15000 } -Expected @(200) -Name "orders/my/{id} PATCH" | Out-Null

    $deleteOrder = New-OrderDraft -BaseUrl $baseUrl -Token $client.AccessToken -Title "Full API Delete Order $stamp" -Budget 1000
    if ($deleteOrder.Ok -and (Require-Value $deleteOrder.Json.id "delete order id")) {
        Invoke-Api -BaseUrl $baseUrl -Method "DELETE" -Path "/api/v1/orders/my/$($deleteOrder.Json.id)" -Token $client.AccessToken -Expected @(200) -Name "orders/my/{id} DELETE" | Out-Null
    }

    $cancelOrder = New-OrderDraft -BaseUrl $baseUrl -Token $client.AccessToken -Title "Full API Cancel Order $stamp" -Budget 1000
    if ($cancelOrder.Ok -and (Require-Value $cancelOrder.Json.id "cancel order id")) {
        Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/orders/my/$($cancelOrder.Json.id)/cancel" -Token $client.AccessToken -Expected @(200) -Name "orders/my/{id}/cancel POST" | Out-Null
    }

    $submitOrder = Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/orders/my/$orderId/submit" -Token $client.AccessToken -Expected @(200) -Name "orders/my/{id}/submit POST"
    if (-not $submitOrder.Ok) {
        return
    }
    Write-Check "main order is published" ($submitOrder.Json.order.status -eq "published") "status=$($submitOrder.Json.order.status)"

    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/orders?page=1&page_size=5&category=tax" -Expected @(200) -Name "orders GET list" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/orders/$orderId" -Expected @(200) -Name "orders/{id} GET public" | Out-Null

    $deleteResponseOrderId = New-PublishedOrder -BaseUrl $baseUrl -Client $client -Title "Full API Delete Response Order $stamp" -Budget 2000
    if ($null -ne $deleteResponseOrderId) {
        $deleteResponse = Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/orders/$deleteResponseOrderId/responses" -Token $executor.AccessToken -Body @{ cover_letter = "Delete this draft response."; proposed_amount = 1500 } -Expected @(201) -Name "orders/{id}/responses POST delete flow"
        if ($deleteResponse.Ok -and (Require-Value $deleteResponse.Json.id "delete response id")) {
            Invoke-Api -BaseUrl $baseUrl -Method "DELETE" -Path "/api/v1/orders/$deleteResponseOrderId/responses/my/$($deleteResponse.Json.id)" -Token $executor.AccessToken -Expected @(200) -Name "orders/{id}/responses/my/{rid} DELETE" | Out-Null
        }
    }

    $failPaymentOrderId = New-PublishedOrder -BaseUrl $baseUrl -Client $client -Title "Full API Fail Payment Order $stamp" -Budget 2000
    if ($null -ne $failPaymentOrderId) {
        $failResponse = Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/orders/$failPaymentOrderId/responses" -Token $executor.AccessToken -Body @{ cover_letter = "Fail this response payment."; proposed_amount = 1500 } -Expected @(201) -Name "orders/{id}/responses POST fail flow"
        if ($failResponse.Ok -and (Require-Value $failResponse.Json.id "fail response id")) {
            $failResponseId = $failResponse.Json.id
            $failSubmit = Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/orders/$failPaymentOrderId/responses/my/$failResponseId/submit" -Token $executor.AccessToken -Expected @(200) -Name "orders/{id}/responses/my/{rid}/submit fail flow"
            if ($failSubmit.Ok -and (Require-Value $failSubmit.Json.payment.transaction_id "fail payment transaction id")) {
                Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/dev/payments/$($failSubmit.Json.payment.transaction_id)/fail" -Token $admin.AccessToken -Expected @(200) -Name "dev/payments/{txId}/fail POST" | Out-Null
            }
        }
    }

    $createResponse = Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/orders/$orderId/responses" -Token $executor.AccessToken -Body @{
        cover_letter    = "I can handle this full API test order."
        proposed_amount = 10000
        currency        = "KZT"
    } -Expected @(201) -Name "orders/{id}/responses POST"
    if (-not $createResponse.Ok -or -not (Require-Value $createResponse.Json.id "main response id")) {
        return
    }
    $responseId = $createResponse.Json.id

    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/orders/$orderId/responses/my" -Token $executor.AccessToken -Expected @(200) -Name "orders/{id}/responses/my GET" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/orders/$orderId/responses/my/$responseId" -Token $executor.AccessToken -Expected @(200) -Name "orders/{id}/responses/my/{rid} GET" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "PATCH" -Path "/api/v1/orders/$orderId/responses/my/$responseId" -Token $executor.AccessToken -Body @{ proposed_amount = 9500 } -Expected @(200) -Name "orders/{id}/responses/my/{rid} PATCH" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/my/responses" -Token $executor.AccessToken -Expected @(200) -Name "my/responses GET list" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/my/responses/$responseId" -Token $executor.AccessToken -Expected @(200) -Name "my/responses/{id} GET" | Out-Null

    $submitResponse = Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/orders/$orderId/responses/my/$responseId/submit" -Token $executor.AccessToken -Expected @(200) -Name "orders/{id}/responses/my/{rid}/submit POST"
    if (-not $submitResponse.Ok -or -not (Require-Value $submitResponse.Json.payment.transaction_id "response payment transaction id")) {
        return
    }
    Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/dev/payments/$($submitResponse.Json.payment.transaction_id)/confirm" -Token $admin.AccessToken -Expected @(200) -Name "dev/payments/{txId}/confirm POST" | Out-Null

    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/client/orders/$orderId/responses" -Token $client.AccessToken -Expected @(200) -Name "client/orders/{id}/responses GET" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/client/orders/$orderId/responses/$responseId" -Token $client.AccessToken -Expected @(200) -Name "client/orders/{id}/responses/{rid} GET" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/client/orders/$orderId/select-response/$responseId" -Token $client.AccessToken -Expected @(200) -Name "client/orders/{id}/select-response/{rid} POST" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/client/orders/$orderId/selection" -Token $client.AccessToken -Expected @(200) -Name "client/orders/{id}/selection GET" | Out-Null

    $myChats = Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/my/chats" -Token $client.AccessToken -Expected @(200) -Name "my/chats GET list"
    $orderChat = Get-FirstItemWhere -ListResult $myChats.Json -Property "order_id" -Value $orderId
    if ($null -ne $orderChat -and (Require-Value $orderChat.chat_id "order chat id")) {
        $chatId = $orderChat.chat_id
        Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/my/chats/$chatId" -Token $client.AccessToken -Expected @(200) -Name "my/chats/{id} GET order" | Out-Null
        $msgCreate = Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/my/chats/$chatId/messages" -Token $client.AccessToken -Body @{ text = "Full API test message from client." } -Expected @(201) -Name "my/chats/{id}/messages POST"
        if ($msgCreate.Ok -and (Require-Value $msgCreate.Json.id "created message id")) {
            $msgId = $msgCreate.Json.id
            Invoke-Api -BaseUrl $baseUrl -Method "PATCH" -Path "/api/v1/my/chats/$chatId/messages/$msgId" -Token $client.AccessToken -Body @{ text = "Full API test message edited." } -Expected @(200) -Name "my/chats/{id}/messages/{mid} PATCH" | Out-Null
            Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/my/chats/$chatId/messages" -Token $executor.AccessToken -Expected @(200) -Name "my/chats/{id}/messages GET" | Out-Null
            Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/my/chats/$chatId/read" -Token $executor.AccessToken -Expected @(200) -Name "my/chats/{id}/read POST" | Out-Null
            Invoke-Api -BaseUrl $baseUrl -Method "DELETE" -Path "/api/v1/my/chats/$chatId/messages/$msgId" -Token $client.AccessToken -Expected @(200) -Name "my/chats/{id}/messages/{mid} DELETE" | Out-Null
        }
        Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/admin/chats" -Token $admin.AccessToken -Expected @(200) -Name "admin/chats GET list" | Out-Null
        Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/admin/chats/$chatId" -Token $admin.AccessToken -Expected @(200) -Name "admin/chats/{id} GET" | Out-Null
        Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/admin/chats/$chatId/messages" -Token $admin.AccessToken -Expected @(200) -Name "admin/chats/{id}/messages GET" | Out-Null
    } else {
        Write-Check "order chat id" $false "chat for order $orderId was not found"
    }

    Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/client/orders/$orderId/complete" -Token $client.AccessToken -Expected @(200) -Name "client/orders/{id}/complete POST" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/client/orders/$orderId/review" -Token $client.AccessToken -Body @{ rating = 5; comment = "Full API test review." } -Expected @(201) -Name "client/orders/{id}/review POST" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/client/orders/$orderId/review" -Token $client.AccessToken -Expected @(200) -Name "client/orders/{id}/review GET" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/client/orders/$orderId/reopen" -Token $client.AccessToken -Expected @(409) -Name "client/orders/{id}/reopen POST after review conflict" | Out-Null

    $myNotifs = Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/my/notifications" -Token $executor.AccessToken -Expected @(200) -Name "my/notifications GET list"
    $notifId = Get-FirstItemId $myNotifs.Json
    if (-not [string]::IsNullOrWhiteSpace($notifId)) {
        Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/my/notifications/$notifId" -Token $executor.AccessToken -Expected @(200) -Name "my/notifications/{id} GET" | Out-Null
        Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/my/notifications/$notifId/read" -Token $executor.AccessToken -Expected @(200) -Name "my/notifications/{id}/read POST" | Out-Null
    }
    Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/my/notifications/read-all" -Token $executor.AccessToken -Expected @(200) -Name "my/notifications/read-all POST" | Out-Null
    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/admin/notifications" -Token $admin.AccessToken -Expected @(200) -Name "admin/notifications GET list" | Out-Null
    if (-not [string]::IsNullOrWhiteSpace($notifId)) {
        Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/admin/notifications/$notifId" -Token $admin.AccessToken -Expected @(200) -Name "admin/notifications/{id} GET" | Out-Null
    }

    Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/my/sanctions" -Token $executor.AccessToken -Expected @(200) -Name "my/sanctions GET list" | Out-Null
    $adminSancs = Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/admin/sanctions" -Token $admin.AccessToken -Expected @(200) -Name "admin/sanctions GET list"
    $sancId = Get-FirstItemId $adminSancs.Json
    if (-not [string]::IsNullOrWhiteSpace($sancId)) {
        Invoke-Api -BaseUrl $baseUrl -Method "GET" -Path "/api/v1/admin/sanctions/$sancId" -Token $admin.AccessToken -Expected @(200) -Name "admin/sanctions/{id} GET" | Out-Null
        Invoke-Api -BaseUrl $baseUrl -Method "POST" -Path "/api/v1/admin/sanctions/$sancId/lift" -Token $admin.AccessToken -Expected @(200) -Name "admin/sanctions/{id}/lift POST" | Out-Null
    }
}

foreach ($baseUrl in $BaseUrls) {
    Run-FullApiTestsForBaseUrl $baseUrl
}

Write-Host ""
Write-Host ("Summary: {0} passed, {1} failed" -f $script:Passed, $script:Failed) -ForegroundColor Cyan

if ($script:Failed -gt 0) {
    exit 1
}
