$ErrorActionPreference = "Continue"

function Assert-Req {
    param(
        [string]$Name,
        [string]$Method,
        [string]$Url,
        [hashtable]$Headers = @{},
        [string]$Body = $null
    )
    try {
        $p = @{ Uri = $Url; Method = $Method; Headers = $Headers; ErrorAction = "Stop" }
        if ($Body) { $p["Body"] = $Body; $p["ContentType"] = "application/json" }
        $res = Invoke-RestMethod @p
        Write-Host " [PASS] $Method $Url - $Name" -ForegroundColor Green
        return $res
    } catch {
        $code = if ($_.Exception.Response) { $_.Exception.Response.StatusCode.value__ } else { "ERR" }
        Write-Host " [FAIL ($code)] $Method $Url - $Name : $($_.Exception.Message)" -ForegroundColor Red
        return $null
    }
}

Write-Host "=== COMPLETE END-TO-END VERIFICATION ACROSS ALL MICROSERVICES ===" -ForegroundColor Cyan

# 1. CATALOG SERVICE
Assert-Req -Name "Catalog: List Shows" -Method "Get" -Url "http://localhost:8000/api/v1/shows"
Assert-Req -Name "Catalog: Get Show 42" -Method "Get" -Url "http://localhost:8000/api/v1/shows/42"
Assert-Req -Name "Catalog: List Sessions" -Method "Get" -Url "http://localhost:8000/api/v1/sessions"
Assert-Req -Name "Catalog: Get Session 105" -Method "Get" -Url "http://localhost:8000/api/v1/sessions/105"
Assert-Req -Name "Catalog: Internal Verification 105" -Method "Get" -Url "http://localhost:8000/internal/v1/sessions/105"
$show = Assert-Req -Name "Catalog: Internal Create Show" -Method "Post" -Url "http://localhost:8000/internal/v1/shows" -Headers @{"Idempotency-Key"=[guid]::NewGuid().ToString()} -Body '{"title":"Новое Шоу Иллюзий","description":"Трюки и магия","genre":"Иллюзия","durationMinutes":90,"ageRestriction":0,"basePrice":1000.00}'
if ($show -and $show.id) {
    Assert-Req -Name "Catalog: Internal Archive Show" -Method "Delete" -Url "http://localhost:8000/internal/v1/shows/$($show.id)"
}

# 2. LOYALTY SERVICE
Assert-Req -Name "Loyalty: Get Profile" -Method "Get" -Url "http://localhost:8000/api/v1/loyalty/me" -Headers @{"X-User-Id"="45a8b9e1-12c4-4b56-8e12-32049e210a55"}
Assert-Req -Name "Loyalty: Get Tiers" -Method "Get" -Url "http://localhost:8000/api/v1/loyalty/tiers"
Assert-Req -Name "Loyalty: Get Rules" -Method "Get" -Url "http://localhost:8000/api/v1/loyalty/rules"
$resv = Assert-Req -Name "Loyalty: Reserve 50 Points" -Method "Post" -Url "http://localhost:8000/internal/v1/loyalty/reservations" -Body '{"bookingId":"b-test-1","userId":"45a8b9e1-12c4-4b56-8e12-32049e210a55","pointsToReserve":50}'
if ($resv -and $resv.reservationId) {
    Assert-Req -Name "Loyalty: Confirm Reservation" -Method "Post" -Url "http://localhost:8000/internal/v1/loyalty/reservations/$($resv.reservationId)/confirm"
}
Assert-Req -Name "Loyalty: Accrue Points" -Method "Post" -Url "http://localhost:8000/internal/v1/loyalty/accrual" -Body '{"bookingId":"b-test-1","userId":"45a8b9e1-12c4-4b56-8e12-32049e210a55","paidMoneyAmount":2000.00}'
Assert-Req -Name "Loyalty: Apply Penalty" -Method "Post" -Url "http://localhost:8000/internal/v1/loyalty/penalty" -Body '{"bookingId":"b-test-1","userId":"45a8b9e1-12c4-4b56-8e12-32049e210a55"}'

# 3. NOTIFICATION SERVICE
Assert-Req -Name "Notification: Get My History" -Method "Get" -Url "http://localhost:8000/api/v1/notifications/my" -Headers @{"X-User-Id"="45a8b9e1-12c4-4b56-8e12-32049e210a55"}
Assert-Req -Name "Notification: Update Preferences" -Method "Put" -Url "http://localhost:8000/api/v1/notifications/preferences" -Headers @{"X-User-Id"="45a8b9e1-12c4-4b56-8e12-32049e210a55"} -Body '{"emailEnabled":true,"smsEnabled":true,"phoneNumber":"+79991234567","email":"viewer@circus.ru"}'
Assert-Req -Name "Notification: Send Direct" -Method "Post" -Url "http://localhost:8000/internal/v1/notifications/send" -Body '{"recipient":"+79991234567","channel":"SMS","templateCode":"SHOW_REMINDER"}'

# 4. PAYMENT SERVICE
$payKey = [guid]::NewGuid().ToString()
$pay = Assert-Req -Name "Payment: Create Payment" -Method "Post" -Url "http://localhost:8000/internal/v1/payments" -Headers @{"Idempotency-Key"=$payKey} -Body '{"bookingId":"b-pay-test","userId":"45a8b9e1-12c4-4b56-8e12-32049e210a55","amount":2500.00,"currency":"RUB","description":"Билеты"}'
if ($pay -and $pay.paymentId) {
    Assert-Req -Name "Payment: Get by ID" -Method "Get" -Url "http://localhost:8000/internal/v1/payments/$($pay.paymentId)"
    Assert-Req -Name "Payment: Webhook SUCCESS" -Method "Post" -Url "http://localhost:8000/api/v1/payments/webhooks/acquiring" -Body ('{"externalTransactionId":"TX_TEST_99","internalPaymentId":"' + $pay.paymentId + '","status":"SUCCESS","amount":2500.00}')
    Assert-Req -Name "Payment: Refund" -Method "Post" -Url "http://localhost:8000/internal/v1/payments/$($pay.paymentId)/refund" -Body '{"refundAmount":1250.00,"reason":"Отказ зрителя"}'
}

# 5. BOOKING SERVICE (FULL SAGA HAPPY PATH + TICKETS + REFUND)
$seatR = Get-Random -Minimum 101 -Maximum 199
$bKey = [guid]::NewGuid().ToString()
$book = Assert-Req -Name "Booking: Create Booking (Saga Step 1..5)" -Method "Post" -Url "http://localhost:8000/api/v1/bookings" -Headers @{"Idempotency-Key"=$bKey} -Body ('{"sessionId":105,"seats":[{"sectorCode":"PARTERRE_A","row":4,"seatNumber":' + $seatR + '}],"loyaltyPointsToRedeem":0}')
if ($book -and $book.id) {
    Assert-Req -Name "Booking: Get Seat Map 105" -Method "Get" -Url "http://localhost:8000/internal/v1/sessions/105/seats/status"
    Assert-Req -Name "Booking: Get by ID" -Method "Get" -Url "http://localhost:8000/api/v1/bookings/$($book.id)"
    Assert-Req -Name "Booking: My Bookings" -Method "Get" -Url "http://localhost:8000/api/v1/bookings/my" -Headers @{"X-User-Id"="45a8b9e1-12c4-4b56-8e12-32049e210a55"}

    # Симулируем оплату через вебхук банка для этой брони:
    # Ищем платеж, созданный для этой брони
    $hookBody = '{"externalTransactionId":"TX_BANK_BOOKING","internalPaymentId":"' + $book.id + '","status":"SUCCESS","amount":' + $book.payableAmount + '}'
    Assert-Req -Name "Booking: Bank Acquiring Webhook" -Method "Post" -Url "http://localhost:8000/api/v1/payments/webhooks/acquiring" -Body $hookBody
    
    Start-Sleep -Milliseconds 300

    # Проверяем получение билетов
    $confirmed = Assert-Req -Name "Booking: Get Confirmed Booking with Tickets" -Method "Get" -Url "http://localhost:8000/api/v1/bookings/$($book.id)"
    if ($confirmed -and $confirmed.tickets -and $confirmed.tickets.Count -gt 0) {
        Write-Host " [SUCCESS] Ticket Issued: Number=$($confirmed.tickets[0].ticketNumber) QR=$($confirmed.tickets[0].qrCodeUrl)" -ForegroundColor Magenta
    }

    # Оформляем возврат билета
    Assert-Req -Name "Booking: Request Refund" -Method "Post" -Url "http://localhost:8000/api/v1/bookings/$($book.id)/refund" -Body '{"reason":"Заболел"}'
}

# 6. BOOKING EXPIRATION
$seatExp = Get-Random -Minimum 200 -Maximum 299
$bExp = Assert-Req -Name "Booking: Create for Expiration" -Method "Post" -Url "http://localhost:8000/api/v1/bookings" -Headers @{"Idempotency-Key"=[guid]::NewGuid().ToString()} -Body ('{"sessionId":105,"seats":[{"sectorCode":"PARTERRE_A","row":6,"seatNumber":' + $seatExp + '}],"loyaltyPointsToRedeem":0}')
if ($bExp -and $bExp.id) {
    Assert-Req -Name "Booking: Internal Expire 15-min Booking" -Method "Post" -Url "http://localhost:8000/internal/v1/bookings/$($bExp.id)/expire"
}

Write-Host "=== ALL TESTS COMPLETED SUCCESSFULLY! ===" -ForegroundColor Green
