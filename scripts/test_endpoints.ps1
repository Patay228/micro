$ErrorActionPreference = "Continue"

function Test-Endpoint {
    param(
        [string]$Name,
        [string]$Method,
        [string]$Url,
        [hashtable]$Headers = @{},
        [string]$Body = $null
    )
    try {
        $params = @{
            Uri = $Url
            Method = $Method
            Headers = $Headers
            ErrorAction = "Stop"
        }
        if ($Body) {
            $params["Body"] = $Body
            $params["ContentType"] = "application/json"
        }
        $resp = Invoke-RestMethod @params
        Write-Host "[PASS] $Method $Url - $Name" -ForegroundColor Green
        return $resp
    } catch {
        $statusCode = "ERR"
        if ($_.Exception.Response) {
            $statusCode = $_.Exception.Response.StatusCode.value__
        }
        Write-Host "[FAIL ($statusCode)] $Method $Url - $Name : $($_.Exception.Message)" -ForegroundColor Red
        return $null
    }
}

Write-Host "===========================================================" -ForegroundColor Cyan
Write-Host " TESTING ALL MICROSERVICE ENDPOINTS VIA API GATEWAY (:8000)" -ForegroundColor Cyan
Write-Host "===========================================================" -ForegroundColor Cyan

# 1. Gateway Health
Test-Endpoint -Name "Gateway Health" -Method "Get" -Url "http://localhost:8000/health"

# 2. Catalog Service
Test-Endpoint -Name "Get All Shows" -Method "Get" -Url "http://localhost:8000/api/v1/shows"
Test-Endpoint -Name "Get Shows with Filter" -Method "Get" -Url "http://localhost:8000/api/v1/shows?genre=Акробатика&ageRestriction=6"
Test-Endpoint -Name "Get Show by ID (42)" -Method "Get" -Url "http://localhost:8000/api/v1/shows/42"
Test-Endpoint -Name "Get All Sessions" -Method "Get" -Url "http://localhost:8000/api/v1/sessions"
Test-Endpoint -Name "Get Session by ID (105)" -Method "Get" -Url "http://localhost:8000/api/v1/sessions/105"
Test-Endpoint -Name "Internal Verify Session (105)" -Method "Get" -Url "http://localhost:8000/internal/v1/sessions/105"

# Create Show (Internal)
$newShowKey = [guid]::NewGuid().ToString()
$createShowBody = '{"title":"Тестовое шоу Мега-Цирк","description":"Новое представление","genre":"Акробатика","durationMinutes":120,"ageRestriction":6,"basePrice":1500.00,"posterUrl":"https://cdn.circus.ru/poster.jpg"}'
$createdShow = Test-Endpoint -Name "Create Show (Internal)" -Method "Post" -Url "http://localhost:8000/internal/v1/shows" -Headers @{"Idempotency-Key"=$newShowKey} -Body $createShowBody
if ($createdShow -and $createdShow.id) {
    Test-Endpoint -Name "Archive Show (Internal)" -Method "Delete" -Url "http://localhost:8000/internal/v1/shows/$($createdShow.id)"
}

# 3. Loyalty Service
Test-Endpoint -Name "Get My Loyalty Profile" -Method "Get" -Url "http://localhost:8000/api/v1/loyalty/me" -Headers @{"X-User-Id"="45a8b9e1-12c4-4b56-8e12-32049e210a55"}
Test-Endpoint -Name "Get Loyalty Tiers" -Method "Get" -Url "http://localhost:8000/api/v1/loyalty/tiers"
Test-Endpoint -Name "Get Marketing Rules" -Method "Get" -Url "http://localhost:8000/api/v1/loyalty/rules"

$resvBody = '{"bookingId":"' + [guid]::NewGuid().ToString() + '","userId":"45a8b9e1-12c4-4b56-8e12-32049e210a55","pointsToReserve":50}'
$resv = Test-Endpoint -Name "Internal Loyalty Reserve Points" -Method "Post" -Url "http://localhost:8000/internal/v1/loyalty/reservations" -Body $resvBody
if ($resv -and $resv.reservationId) {
    Test-Endpoint -Name "Internal Loyalty Release Reservation" -Method "Delete" -Url "http://localhost:8000/internal/v1/loyalty/reservations/$($resv.reservationId)"
}

# 4. Notification Service
Test-Endpoint -Name "Get My Notifications" -Method "Get" -Url "http://localhost:8000/api/v1/notifications/my" -Headers @{"X-User-Id"="45a8b9e1-12c4-4b56-8e12-32049e210a55"}
$prefBody = '{"emailEnabled":true,"smsEnabled":true,"phoneNumber":"+79991234567","email":"test@circus.ru"}'
Test-Endpoint -Name "Update Notification Preferences" -Method "Put" -Url "http://localhost:8000/api/v1/notifications/preferences" -Headers @{"X-User-Id"="45a8b9e1-12c4-4b56-8e12-32049e210a55"} -Body $prefBody
$sendBody = '{"recipient":"+79991234567","channel":"SMS","templateCode":"TICKET_INFO","params":{"showTitle":"Магия цирка"}}'
Test-Endpoint -Name "Internal Send Notification" -Method "Post" -Url "http://localhost:8000/internal/v1/notifications/send" -Body $sendBody

# 5. Payment Service
$payKey = [guid]::NewGuid().ToString()
$payBody = '{"bookingId":"' + [guid]::NewGuid().ToString() + '","userId":"45a8b9e1-12c4-4b56-8e12-32049e210a55","amount":2500.00,"currency":"RUB","description":"Билеты на цирк"}'
$payObj = Test-Endpoint -Name "Internal Create Payment" -Method "Post" -Url "http://localhost:8000/internal/v1/payments" -Headers @{"Idempotency-Key"=$payKey} -Body $payBody
if ($payObj -and $payObj.paymentId) {
    Test-Endpoint -Name "Internal Get Payment by ID" -Method "Get" -Url "http://localhost:8000/internal/v1/payments/$($payObj.paymentId)"
    $webhookBody = '{"externalTransactionId":"TX_BANK_' + (Get-Random) + '","internalPaymentId":"' + $payObj.paymentId + '","status":"SUCCESS","amount":2500.00}'
    Test-Endpoint -Name "Public Acquiring Webhook" -Method "Post" -Url "http://localhost:8000/api/v1/payments/webhooks/acquiring" -Body $webhookBody
}

# 6. Booking Service
$seatNum = Get-Random -Minimum 30 -Maximum 99
$bookKey = [guid]::NewGuid().ToString()
$bookBody = '{"sessionId":105,"seats":[{"sectorCode":"PARTERRE_A","row":5,"seatNumber":' + $seatNum + '}],"loyaltyPointsToRedeem":50}'
$booked = Test-Endpoint -Name "Create Booking (Saga Happy Path)" -Method "Post" -Url "http://localhost:8000/api/v1/bookings" -Headers @{"Idempotency-Key"=$bookKey} -Body $bookBody
if ($booked -and $booked.id) {
    Test-Endpoint -Name "Get Booking by ID" -Method "Get" -Url "http://localhost:8000/api/v1/bookings/$($booked.id)"
    Test-Endpoint -Name "Get My Bookings" -Method "Get" -Url "http://localhost:8000/api/v1/bookings/my" -Headers @{"X-User-Id"="45a8b9e1-12c4-4b56-8e12-32049e210a55"}
    Test-Endpoint -Name "Get Seat Status for Session 105" -Method "Get" -Url "http://localhost:8000/internal/v1/sessions/105/seats/status"
    Test-Endpoint -Name "Cancel Booking" -Method "Post" -Url "http://localhost:8000/api/v1/bookings/$($booked.id)/cancel"
}
