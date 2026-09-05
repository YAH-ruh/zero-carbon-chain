# ============================================================
# Micro Carbon Chain - Full API Test Script (PowerShell)
# Usage: Start the server first (weitanlian.exe), then run:
#   .\test\test_api.ps1
# ============================================================

$BASE_URL = "http://localhost:8080/api"
$PASS = 0
$FAIL = 0
$GLOBAL_TOKENS = @{}

# ============================================================
# Helper Functions
# ============================================================
function Write-TestHeader {
    param([string]$Title)
    Write-Host "`n========================================" -ForegroundColor Cyan
    Write-Host "  $Title" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
}

function Write-TestResult {
    param([string]$Name, [bool]$Success, [string]$Detail)
    if ($Success) {
        $script:PASS++
        Write-Host "  [PASS] $Name" -ForegroundColor Green
    } else {
        $script:FAIL++
        Write-Host "  [FAIL] $Name - $Detail" -ForegroundColor Red
    }
}

function Invoke-Api {
    param(
        [string]$Method = "GET",
        [string]$Path,
        [string]$Body = $null,
        [string]$Token = $null
    )
    $uri = "$BASE_URL$Path"
    $params = @{
        Uri = $uri
        Method = $Method
        ContentType = "application/json"
        UseBasicParsing = $true
    }
    if ($Body) { $params.Body = $Body }
    if ($Token) { $params.Headers = @{ Authorization = "Bearer $Token" } }
    try {
        $resp = Invoke-WebRequest @params
        $content = $resp.Content | ConvertFrom-Json
        return @{ Success = $true; Data = $content; Raw = $resp.Content }
    } catch {
        $errMsg = $_.Exception.Message
        try {
            $errContent = $_.ErrorDetails.Message | ConvertFrom-Json
            $errMsg = $errContent.msg
        } catch {}
        return @{ Success = $false; Error = $errMsg }
    }
}

# ============================================================
# 1. Health Check
# ============================================================
Write-TestHeader "1. Health Check"

$r = Invoke-Api -Method GET -Path "/health"
Write-TestResult "Health Check" $r.Success ($r.Error)
if ($r.Success) { Write-Host "    Service: $($r.Data.service)" -ForegroundColor Gray }

# ============================================================
# 2. User Registration (4 roles)
# ============================================================
Write-TestHeader "2. User Registration (4 roles)"

$users = @(
    @{ username = "ent_test"; password = "123456"; company = "GreenTech"; role = "enterprise"; park_id = 1 },
    @{ username = "park_test"; password = "123456"; company = "ParkAdmin"; role = "park_admin"; park_id = 1 },
    @{ username = "exch_test"; password = "123456"; company = "CarbonExch"; role = "exchange"; park_id = 0 },
    @{ username = "reg_test"; password = "123456"; company = "Regulator"; role = "regulator"; park_id = 0 }
)

foreach ($u in $users) {
    $body = $u | ConvertTo-Json -Compress
    $r = Invoke-Api -Method POST -Path "/auth/register" -Body $body
    $ok = $r.Success -or ($r.Data.msg -match "exists")
    Write-TestResult "Register [$($u.role)] $($u.username)" $ok $r.Error
}

# ============================================================
# 3. Login & Get Tokens
# ============================================================
Write-TestHeader "3. Login & Get Tokens"

$loginUsers = @(
    @{ key = "admin";  username = "admin";      password = "admin123" },
    @{ key = "ent";    username = "ent_test";   password = "123456" },
    @{ key = "park";   username = "park_test";  password = "123456" },
    @{ key = "exch";   username = "exch_test";  password = "123456" },
    @{ key = "reg";    username = "reg_test";   password = "123456" }
)

foreach ($u in $loginUsers) {
    $body = "{`"username`":`"$($u.username)`",`"password`":`"$($u.password)`"}"
    $r = Invoke-Api -Method POST -Path "/auth/login" -Body $body
    if ($r.Success) {
        $GLOBAL_TOKENS[$u.key] = $r.Data.data.token
        Write-TestResult "Login [$($u.key)] $($u.username)" $true ""
        Write-Host "    Role: $($r.Data.data.user.role)" -ForegroundColor Gray
    } else {
        Write-TestResult "Login [$($u.key)] $($u.username)" $false $r.Error
    }
}

# ============================================================
# 4. Get Current User Info
# ============================================================
Write-TestHeader "4. Get Current User Info"

$r = Invoke-Api -Method GET -Path "/auth/me" -Token $GLOBAL_TOKENS["admin"]
Write-TestResult "Get Current User Info" $r.Success $r.Error
if ($r.Success) { Write-Host "    User: $($r.Data.data.username), Role: $($r.Data.data.role)" -ForegroundColor Gray }

# Get enterprise user ID
$r = Invoke-Api -Method GET -Path "/auth/me" -Token $GLOBAL_TOKENS["ent"]
$entId = if ($r.Success) { $r.Data.data.id } else { 2 }

# ============================================================
# 5. Simulate Energy Data
# ============================================================
Write-TestHeader "5. Simulate Energy Data"

$body = "{`"enterprise_id`":$entId,`"device_id`":`"METER-TEST-001`",`"count`":3}"
$r = Invoke-Api -Method POST -Path "/energy/simulate" -Body $body -Token $GLOBAL_TOKENS["ent"]
Write-TestResult "Simulate Energy Data (3 records)" $r.Success $r.Error
if ($r.Success) { Write-Host "    Generated: $($r.Data.data.Count) records" -ForegroundColor Gray }

# Query energy records
$r = Invoke-Api -Method GET -Path "/energy/list?enterprise_id=$entId&page=1&page_size=10" -Token $GLOBAL_TOKENS["ent"]
Write-TestResult "Query Energy Records" $r.Success $r.Error
if ($r.Success) { Write-Host "    Total: $($r.Data.data.total), Current page: $($r.Data.data.list.Count) records" -ForegroundColor Gray }

$firstEnergyId = if ($r.Success -and $r.Data.data.list.Count -gt 0) { $r.Data.data.list[0].id } else { 1 }
$firstEnergyNo = if ($r.Success -and $r.Data.data.list.Count -gt 0) { $r.Data.data.list[0].record_no } else { "" }

# ============================================================
# 6. Calculate Carbon Credits & Upload to Chain
# ============================================================
Write-TestHeader "6. Calculate Carbon Credits & Upload to Chain"

$body = "{`"energy_record_id`":$firstEnergyId}"
$r = Invoke-Api -Method POST -Path "/carbon/calculate" -Body $body -Token $GLOBAL_TOKENS["ent"]
Write-TestResult "Calculate Carbon Credits & Upload" $r.Success $r.Error
if ($r.Success) {
    Write-Host "    Credit No: $($r.Data.data.credit.credit_no)" -ForegroundColor Gray
    Write-Host "    Emission: $([math]::Round($r.Data.data.credit.total_emission, 2)) kgCO2" -ForegroundColor Gray
    Write-Host "    Credits: $([math]::Round($r.Data.data.credit.carbon_credits, 2))" -ForegroundColor Gray
    Write-Host "    Block Hash: $($r.Data.data.block_hash)" -ForegroundColor Gray
    $creditNo = $r.Data.data.credit.credit_no
}

# ============================================================
# 7. Query Carbon Credits
# ============================================================
Write-TestHeader "7. Query Carbon Credits"

$r = Invoke-Api -Method GET -Path "/carbon/my-credits?enterprise_id=$entId" -Token $GLOBAL_TOKENS["ent"]
Write-TestResult "List Carbon Credits" $r.Success $r.Error
if ($r.Success) { Write-Host "    Total: $($r.Data.data.total)" -ForegroundColor Gray }

$r = Invoke-Api -Method GET -Path "/carbon/stats?enterprise_id=$entId" -Token $GLOBAL_TOKENS["ent"]
Write-TestResult "Carbon Stats" $r.Success $r.Error
if ($r.Success) {
    Write-Host "    Total Emission: $([math]::Round($r.Data.data.total_emission, 2)) kgCO2" -ForegroundColor Gray
    Write-Host "    Total Credits: $([math]::Round($r.Data.data.total_credits, 2))" -ForegroundColor Gray
}

# ============================================================
# 8. Blockchain Functionality
# ============================================================
Write-TestHeader "8. Blockchain Functionality"

# Chain info
$r = Invoke-Api -Method GET -Path "/chain/info" -Token $GLOBAL_TOKENS["ent"]
Write-TestResult "Get Blockchain Info" $r.Success $r.Error
if ($r.Success) { Write-Host "    Block count: $($r.Data.data.block_count)" -ForegroundColor Gray }

# Block list
$r = Invoke-Api -Method GET -Path "/chain/blocks?page=1&page_size=20" -Token $GLOBAL_TOKENS["ent"]
Write-TestResult "Get Block List" $r.Success $r.Error
if ($r.Success) { Write-Host "    Total blocks: $($r.Data.data.total)" -ForegroundColor Gray }

# Trace query
if ($creditNo) {
    $r = Invoke-Api -Method GET -Path "/chain/query?data_no=$creditNo" -Token $GLOBAL_TOKENS["ent"]
    Write-TestResult "Trace Query ($creditNo)" $r.Success $r.Error
    if ($r.Success) { Write-Host "    Block index: $($r.Data.data.block_index), Match: $($r.Data.data.match)" -ForegroundColor Gray }
}

# Tamper verification
if ($creditNo) {
    $body = "{`"data_type`":`"credit`",`"data_id`":`"$creditNo`"}"
    $r = Invoke-Api -Method POST -Path "/chain/verify" -Body $body -Token $GLOBAL_TOKENS["ent"]
    Write-TestResult "Tamper Verification ($creditNo)" $r.Success $r.Error
    if ($r.Success) { Write-Host "    Result: $($r.Data.msg)" -ForegroundColor Gray }
}

# Upload to chain
if ($firstEnergyNo) {
    $body = "{`"data_type`":`"energy`",`"data_id`":`"$firstEnergyNo`"}"
    $r = Invoke-Api -Method POST -Path "/chain/upload" -Body $body -Token $GLOBAL_TOKENS["ent"]
    Write-TestResult "Upload to Chain ($firstEnergyNo)" $r.Success $r.Error
}

# ============================================================
# 9. Sell Order
# ============================================================
Write-TestHeader "9. Sell Order"

# Find available credit
$r = Invoke-Api -Method GET -Path "/carbon/my-credits?enterprise_id=$entId" -Token $GLOBAL_TOKENS["ent"]
$creditId = 0
if ($r.Success -and $r.Data.data.list.Count -gt 0) {
    foreach ($c in $r.Data.data.list) {
        if ($c.status -eq "available") { $creditId = $c.id; break }
    }
}

if ($creditId -gt 0) {
    $body = "{`"credit_id`":$creditId,`"quantity`":50,`"unit_price`":12.5}"
    $r = Invoke-Api -Method POST -Path "/carbon/sell" -Body $body -Token $GLOBAL_TOKENS["ent"]
    Write-TestResult "Create Sell Order" $r.Success $r.Error
    if ($r.Success) {
        Write-Host "    Order No: $($r.Data.data.order_no), Status: $($r.Data.data.status)" -ForegroundColor Gray
    }
} else {
    Write-TestResult "Create Sell Order" $false "No available credits"
}

# List sell orders
$r = Invoke-Api -Method GET -Path "/carbon/sell-orders?status=pending" -Token $GLOBAL_TOKENS["ent"]
Write-TestResult "List Sell Orders" $r.Success $r.Error
if ($r.Success) { Write-Host "    Total: $($r.Data.data.total)" -ForegroundColor Gray }

# ============================================================
# 10. Park Admin Functionality
# ============================================================
Write-TestHeader "10. Park Admin Functionality"

$r = Invoke-Api -Method GET -Path "/admin/park/overview?park_id=1" -Token $GLOBAL_TOKENS["park"]
Write-TestResult "Park Carbon Overview" $r.Success $r.Error
if ($r.Success) { Write-Host "    Enterprises: $($r.Data.data.enterprise_count), Total Emission: $([math]::Round($r.Data.data.total_emission, 2))" -ForegroundColor Gray }

$r = Invoke-Api -Method GET -Path "/admin/park/enterprises?park_id=1" -Token $GLOBAL_TOKENS["park"]
Write-TestResult "Park Enterprise List" $r.Success $r.Error

$r = Invoke-Api -Method GET -Path "/admin/enterprise/detail?enterprise_id=$entId" -Token $GLOBAL_TOKENS["park"]
Write-TestResult "Enterprise Detail" $r.Success $r.Error

# Generate park report
$body = "{`"park_id`":1,`"park_name`":`"Green Tech Demo Park`"}"
$r = Invoke-Api -Method POST -Path "/admin/park/report" -Body $body -Token $GLOBAL_TOKENS["park"]
Write-TestResult "Generate Park Report" $r.Success $r.Error
if ($r.Success) { Write-Host "    Report preview: $($r.Data.data.report.Substring(0, [Math]::Min(80, $r.Data.data.report.Length)))..." -ForegroundColor Gray }

# ============================================================
# 11. Exchange Functionality
# ============================================================
Write-TestHeader "11. Exchange Functionality"

$r = Invoke-Api -Method GET -Path "/exchange/orders" -Token $GLOBAL_TOKENS["exch"]
Write-TestResult "Browse Pending Orders" $r.Success $r.Error
if ($r.Success) { Write-Host "    Total: $($r.Data.data.total)" -ForegroundColor Gray }

# Try to match an order
$orderId = 0
if ($r.Success -and $r.Data.data.list.Count -gt 0) { $orderId = $r.Data.data.list[0].id }
if ($orderId -gt 0) {
    $body = "{`"order_id`":$orderId,`"buyer_id`":$entId}"
    $r = Invoke-Api -Method POST -Path "/exchange/match" -Body $body -Token $GLOBAL_TOKENS["exch"]
    Write-TestResult "Match Order" $r.Success $r.Error
    if ($r.Success) {
        Write-Host "    Tx No: $($r.Data.data.transaction.tx_no)" -ForegroundColor Gray
        Write-Host "    Amount: $($r.Data.data.transaction.total_amount) yuan" -ForegroundColor Gray
    }
} else {
    Write-TestResult "Match Order" $false "No pending orders (skipped)"
}

# Verify credit source
if ($creditNo) {
    $body = "{`"credit_no`":`"$creditNo`"}"
    $r = Invoke-Api -Method POST -Path "/exchange/verify-credit" -Body $body -Token $GLOBAL_TOKENS["exch"]
    Write-TestResult "Verify Credit Source" $r.Success $r.Error
    if ($r.Success) { Write-Host "    Result: $($r.Data.msg)" -ForegroundColor Gray }
}

# Transaction records
$r = Invoke-Api -Method GET -Path "/exchange/transactions" -Token $GLOBAL_TOKENS["exch"]
Write-TestResult "List Transactions" $r.Success $r.Error
if ($r.Success) { Write-Host "    Total: $($r.Data.data.total)" -ForegroundColor Gray }

# ============================================================
# 12. Enterprise AI Advice
# ============================================================
Write-TestHeader "12. Enterprise AI Advice"

$body = "{`"enterprise_id`":$entId,`"company_name`":`"GreenTech`",`"industry`":`"Manufacturing`"}"
$r = Invoke-Api -Method POST -Path "/enterprise/advice" -Body $body -Token $GLOBAL_TOKENS["ent"]
Write-TestResult "Generate Emission Reduction Advice" $r.Success $r.Error
if ($r.Success) { Write-Host "    Advice preview: $($r.Data.data.advice.Substring(0, [Math]::Min(80, $r.Data.data.advice.Length)))..." -ForegroundColor Gray }

# ============================================================
# 13. Regulator Functionality
# ============================================================
Write-TestHeader "13. Regulator Functionality"

$r = Invoke-Api -Method GET -Path "/regulator/chain/all?page=1&page_size=20" -Token $GLOBAL_TOKENS["reg"]
Write-TestResult "All Chain Records" $r.Success $r.Error
if ($r.Success) { Write-Host "    Total blocks: $($r.Data.data.total)" -ForegroundColor Gray }

$r = Invoke-Api -Method GET -Path "/regulator/platform/stats" -Token $GLOBAL_TOKENS["reg"]
Write-TestResult "Platform Stats" $r.Success $r.Error
if ($r.Success) {
    Write-Host "    Issued Credits: $([math]::Round($r.Data.data.total_issued_credits, 2))" -ForegroundColor Gray
    Write-Host "    Traded Credits: $([math]::Round($r.Data.data.total_traded_credits, 2))" -ForegroundColor Gray
    Write-Host "    Enterprises: $($r.Data.data.enterprise_count)" -ForegroundColor Gray
    Write-Host "    Transactions: $($r.Data.data.transaction_count)" -ForegroundColor Gray
}

$r = Invoke-Api -Method GET -Path "/regulator/risk-alert" -Token $GLOBAL_TOKENS["reg"]
Write-TestResult "Risk Alert" $r.Success $r.Error
if ($r.Success) {
    Write-Host "    Alert count: $($r.Data.data.alertCount)" -ForegroundColor Gray
    foreach ($a in $r.Data.data.alerts) {
        Write-Host "    [$($a.level)] $($a.message)" -ForegroundColor Yellow
    }
}

$r = Invoke-Api -Method GET -Path "/regulator/users" -Token $GLOBAL_TOKENS["reg"]
Write-TestResult "All Users" $r.Success $r.Error
if ($r.Success) { Write-Host "    Total users: $($r.Data.data.total)" -ForegroundColor Gray }

# Regulator verify data
if ($creditNo) {
    $body = "{`"data_type`":`"credit`",`"data_id`":`"$creditNo`"}"
    $r = Invoke-Api -Method POST -Path "/regulator/verify" -Body $body -Token $GLOBAL_TOKENS["reg"]
    Write-TestResult "Regulator Verify Data" $r.Success $r.Error
    if ($r.Success) { Write-Host "    Result: $($r.Data.msg)" -ForegroundColor Gray }
}

# ============================================================
# 14. Park Admin - Enterprise Status Management
# ============================================================
Write-TestHeader "14. Park Admin - Enterprise Status Management"

$body = "{`"enterprise_id`":$entId,`"status`":1}"
$r = Invoke-Api -Method PUT -Path "/admin/enterprise/status" -Body $body -Token $GLOBAL_TOKENS["park"]
Write-TestResult "Update Enterprise Status" $r.Success $r.Error
if ($r.Success) { Write-Host "    Result: $($r.Data.msg)" -ForegroundColor Gray }

# ============================================================
# Summary
# ============================================================
Write-TestHeader "Test Summary"
$total = $PASS + $FAIL
Write-Host "  Total:    $total" -ForegroundColor White
Write-Host "  Passed:   $PASS" -ForegroundColor Green
Write-Host "  Failed:   $FAIL" -ForegroundColor Red
$rate = if ($total -gt 0) { [math]::Round($PASS / $total * 100, 1) } else { 0 }
Write-Host "  Rate:     $rate%" -ForegroundColor White

if ($FAIL -eq 0) {
    Write-Host "`nAll tests passed!" -ForegroundColor Green
} else {
    Write-Host "`nSome tests failed. Please check server logs." -ForegroundColor Yellow
}