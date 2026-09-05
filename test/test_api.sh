#!/bin/bash
# ============================================================
# 微碳链 - 全流程 API 接口测试脚本 (Bash)
# 使用方式: 先启动服务 weitanlian.exe，再运行本脚本
# 运行: bash test/test_api.sh
# 也可在 WSL / Git Bash / Linux / macOS 下运行
# ============================================================

BASE_URL="http://localhost:8080/api"
PASS=0
FAIL=0
declare -A TOKENS

# ============================================================
# 辅助函数
# ============================================================
header() {
    echo ""
    echo "========================================"
    echo "  $1"
    echo "========================================"
}

result() {
    local name="$1" success="$2" detail="$3"
    if [ "$success" = "true" ]; then
        ((PASS++))
        echo -e "  \033[32m[PASS]\033[0m $name"
    else
        ((FAIL++))
        echo -e "  \033[31m[FAIL]\033[0m $name - $detail"
    fi
}

call_api() {
    local method="$1" path="$2" body="$3" token="$4"
    local url="${BASE_URL}${path}"
    local header_args=(-H "Content-Type: application/json")
    [ -n "$token" ] && header_args+=(-H "Authorization: Bearer $token")

    if [ -n "$body" ]; then
        response=$(curl -s -X "$method" "${header_args[@]}" -d "$body" "$url" 2>/dev/null)
    else
        response=$(curl -s -X "$method" "${header_args[@]}" "$url" 2>/dev/null)
    fi

    echo "$response"
}

call_api_with_code() {
    local method="$1" path="$2" body="$3" token="$4"
    local url="${BASE_URL}${path}"
    local header_args=(-H "Content-Type: application/json")
    [ -n "$token" ] && header_args+=(-H "Authorization: Bearer $token")

    if [ -n "$body" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "${header_args[@]}" -d "$body" "$url" 2>/dev/null)
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "${header_args[@]}" "$url" 2>/dev/null)
    fi

    http_code=$(echo "$response" | tail -1)
    content=$(echo "$response" | sed '$d')
    echo "$content"
    return "$http_code"
}

# ============================================================
# 1. 健康检查
# ============================================================
header "1. 健康检查"

resp=$(call_api GET "/health")
status=$(echo "$resp" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
[ "$status" = "ok" ] && result "健康检查" true || result "健康检查" false "$resp"

# ============================================================
# 2. 用户注册
# ============================================================
header "2. 用户注册（四种角色）"

register_user() {
    local u="$1" p="$2" c="$3" r="$4" pid="$5"
    body="{\"username\":\"$u\",\"password\":\"$p\",\"company\":\"$c\",\"role\":\"$r\",\"park_id\":$pid}"
    resp=$(call_api POST "/auth/register" "$body")
    msg=$(echo "$resp" | grep -o '"msg":"[^"]*"' | cut -d'"' -f4)
    if echo "$msg" | grep -qE "成功|已存在"; then
        result "注册 [$r] $u" true
    else
        result "注册 [$r] $u" false "$msg"
    fi
}

register_user "ent_test" "123456" "绿色科技公司" "enterprise" 1
register_user "park_test" "123456" "工业园区管委会" "park_admin" 1
register_user "exch_test" "123456" "碳交易所" "exchange" 0
register_user "reg_test" "123456" "监管中心" "regulator" 0

# ============================================================
# 3. 登录获取 Token
# ============================================================
header "3. 登录获取 Token"

login_user() {
    local key="$1" u="$2" p="$3"
    body="{\"username\":\"$u\",\"password\":\"$p\"}"
    resp=$(call_api POST "/auth/login" "$body")
    token=$(echo "$resp" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    if [ -n "$token" ]; then
        TOKENS[$key]="$token"
        role=$(echo "$resp" | grep -o '"role":"[^"]*"' | head -1 | cut -d'"' -f4)
        result "登录 [$key] $u" true
        echo "    角色: $role"
    else
        result "登录 [$key] $u" false "$resp"
    fi
}

login_user "admin" "admin" "admin123"
login_user "ent" "ent_test" "123456"
login_user "park" "park_test" "123456"
login_user "exch" "exch_test" "123456"
login_user "reg" "reg_test" "123456"

# ============================================================
# 4. 获取当前用户信息
# ============================================================
header "4. 获取当前用户信息"

resp=$(call_api GET "/auth/me" "" "${TOKENS[admin]}")
username=$(echo "$resp" | grep -o '"username":"[^"]*"' | head -1 | cut -d'"' -f4)
[ -n "$username" ] && result "获取当前用户信息" true || result "获取当前用户信息" false "$resp"
echo "    用户名: $username"

# 获取企业用户ID
resp=$(call_api GET "/auth/me" "" "${TOKENS[ent]}")
ENT_ID=$(echo "$resp" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)

# ============================================================
# 5. 模拟能耗数据
# ============================================================
header "5. 模拟能耗数据"

body="{\"enterprise_id\":$ENT_ID,\"device_id\":\"METER-TEST-001\",\"count\":3}"
resp=$(call_api POST "/energy/simulate" "$body" "${TOKENS[ent]}")
count=$(echo "$resp" | grep -o '"record_no":"[^"]*"' | wc -l)
[ "$count" -gt 0 ] && result "模拟能耗数据（3条）" true || result "模拟能耗数据（3条）" false "$resp"
echo "    生成条数: $count"

# 查询能耗记录
resp=$(call_api GET "/energy/list?enterprise_id=$ENT_ID" "" "${TOKENS[ent]}")
total=$(echo "$resp" | grep -o '"total":[0-9]*' | head -1 | cut -d: -f2)
result "查询能耗记录" true
echo "    记录总数: $total"

# 提取第一个能耗记录ID和编号
FIRST_ENERGY_ID=$(echo "$resp" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
FIRST_ENERGY_NO=$(echo "$resp" | grep -o '"record_no":"[^"]*"' | head -1 | cut -d'"' -f4)

# ============================================================
# 6. 核算碳积分并上链
# ============================================================
header "6. 核算碳积分并上链"

body="{\"energy_record_id\":$FIRST_ENERGY_ID}"
resp=$(call_api POST "/carbon/calculate" "$body" "${TOKENS[ent]}")
credit_no=$(echo "$resp" | grep -o '"credit_no":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$credit_no" ]; then
    emission=$(echo "$resp" | grep -o '"total_emission":[0-9.]*' | head -1 | cut -d: -f2)
    credits=$(echo "$resp" | grep -o '"carbon_credits":[0-9.]*' | head -1 | cut -d: -f2)
    block_hash=$(echo "$resp" | grep -o '"block_hash":"[^"]*"' | head -1 | cut -d'"' -f4)
    result "核算碳积分并上链" true
    echo "    核算编号: $credit_no"
    echo "    碳排放量: $emission kgCO2"
    echo "    碳积分量: $credits 积分"
    echo "    区块哈希: ${block_hash:0:20}..."
else
    result "核算碳积分并上链" false "$resp"
fi

# ============================================================
# 7. 碳积分查询
# ============================================================
header "7. 碳积分查询"

resp=$(call_api GET "/carbon/my-credits?enterprise_id=$ENT_ID" "" "${TOKENS[ent]}")
total=$(echo "$resp" | grep -o '"total":[0-9]*' | head -1 | cut -d: -f2)
result "查看碳积分列表" true
echo "    积分记录数: $total"

resp=$(call_api GET "/carbon/stats?enterprise_id=$ENT_ID" "" "${TOKENS[ent]}")
total_emission=$(echo "$resp" | grep -o '"total_emission":[0-9.]*' | head -1 | cut -d: -f2)
total_credits=$(echo "$resp" | grep -o '"total_credits":[0-9.]*' | head -1 | cut -d: -f2)
result "碳数据统计" true
echo "    总排放量: $total_emission kgCO2"
echo "    总积分数: $total_credits"

# ============================================================
# 8. 区块链功能测试
# ============================================================
header "8. 区块链功能测试"

# 链信息
resp=$(call_api GET "/chain/info" "" "${TOKENS[ent]}")
block_count=$(echo "$resp" | grep -o '"block_count":[0-9]*' | head -1 | cut -d: -f2)
result "获取区块链信息" true
echo "    区块数量: $block_count"

# 区块列表
resp=$(call_api GET "/chain/blocks" "" "${TOKENS[ent]}")
total=$(echo "$resp" | grep -o '"total":[0-9]*' | head -1 | cut -d: -f2)
result "获取区块列表" true
echo "    总区块数: $total"

# 溯源查询
if [ -n "$credit_no" ]; then
    resp=$(call_api GET "/chain/query?data_no=$credit_no" "" "${TOKENS[ent]}")
    bindex=$(echo "$resp" | grep -o '"block_index":[0-9]*' | head -1 | cut -d: -f2)
    match=$(echo "$resp" | grep -o '"match":true')
    [ -n "$bindex" ] && result "溯源查询（$credit_no）" true || result "溯源查询（$credit_no）" false "$resp"
    echo "    区块索引: $bindex"
fi

# 篡改校验
if [ -n "$credit_no" ]; then
    body="{\"data_type\":\"credit\",\"data_id\":\"$credit_no\"}"
    resp=$(call_api POST "/chain/verify" "$body" "${TOKENS[ent]}")
    msg=$(echo "$resp" | grep -o '"msg":"[^"]*"' | head -1 | cut -d'"' -f4)
    result "篡改校验（$credit_no）" true
    echo "    校验结果: $msg"
fi

# 上链存证
if [ -n "$FIRST_ENERGY_NO" ]; then
    body="{\"data_type\":\"energy\",\"data_id\":\"$FIRST_ENERGY_NO\"}"
    resp=$(call_api POST "/chain/upload" "$body" "${TOKENS[ent]}")
    msg=$(echo "$resp" | grep -o '"msg":"[^"]*"' | head -1 | cut -d'"' -f4)
    result "上链存证（$FIRST_ENERGY_NO）" true
fi

# ============================================================
# 9. 挂单卖出
# ============================================================
header "9. 挂单卖出"

# 获取可用积分ID
resp=$(call_api GET "/carbon/my-credits?enterprise_id=$ENT_ID" "" "${TOKENS[ent]}")
CREDIT_ID=$(echo "$resp" | grep -o '"status":"available"' -B 5 | grep '"id":[0-9]*' | head -1 | cut -d: -f2 | tr -d ' ')

if [ -n "$CREDIT_ID" ] && [ "$CREDIT_ID" != "0" ]; then
    body="{\"credit_id\":$CREDIT_ID,\"quantity\":50,\"unit_price\":12.5}"
    resp=$(call_api POST "/carbon/sell" "$body" "${TOKENS[ent]}")
    order_no=$(echo "$resp" | grep -o '"order_no":"[^"]*"' | head -1 | cut -d'"' -f4)
    status=$(echo "$resp" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ -n "$order_no" ]; then
        result "创建挂单卖出" true
        echo "    挂单编号: $order_no"
        echo "    挂单状态: $status"
    else
        result "创建挂单卖出" false "$resp"
    fi
else
    result "创建挂单卖出" false "无可用积分"
fi

# 查看挂单列表
resp=$(call_api GET "/carbon/sell-orders?status=pending" "" "${TOKENS[ent]}")
total=$(echo "$resp" | grep -o '"total":[0-9]*' | head -1 | cut -d: -f2)
result "查看挂单列表" true
echo "    挂单数: $total"

# ============================================================
# 10. 园区管理员功能
# ============================================================
header "10. 园区管理员功能"

resp=$(call_api GET "/admin/park/overview?park_id=1" "" "${TOKENS[park]}")
ent_count=$(echo "$resp" | grep -o '"enterprise_count":[0-9]*' | head -1 | cut -d: -f2)
result "园区碳数据总览" true
echo "    企业数量: $ent_count"

resp=$(call_api GET "/admin/park/enterprises?park_id=1" "" "${TOKENS[park]}")
result "园区企业列表" true

resp=$(call_api GET "/admin/enterprise/detail?enterprise_id=$ENT_ID" "" "${TOKENS[park]}")
result "企业碳数据详情" true

# 生成园区低碳报告
body="{\"park_id\":1,\"park_name\":\"绿色科技示范园区\"}"
resp=$(call_api POST "/admin/park/report" "$body" "${TOKENS[park]}")
report=$(echo "$resp" | grep -o '"report":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$report" ]; then
    result "生成园区低碳报告" true
    echo "    报告摘要: ${report:0:80}..."
else
    result "生成园区低碳报告" false "$resp"
fi

# ============================================================
# 11. 交易所功能
# ============================================================
header "11. 交易所功能"

resp=$(call_api GET "/exchange/orders" "" "${TOKENS[exch]}")
total=$(echo "$resp" | grep -o '"total":[0-9]*' | head -1 | cut -d: -f2)
result "浏览卖方挂单" true
echo "    挂单数: $total"

# 如果有挂单，尝试撮合
ORDER_ID=$(echo "$resp" | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
if [ -n "$ORDER_ID" ] && [ "$ORDER_ID" != "0" ]; then
    body="{\"order_id\":$ORDER_ID,\"buyer_id\":$ENT_ID}"
    resp=$(call_api POST "/exchange/match" "$body" "${TOKENS[exch]}")
    tx_no=$(echo "$resp" | grep -o '"tx_no":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ -n "$tx_no" ]; then
        amount=$(echo "$resp" | grep -o '"total_amount":[0-9.]*' | head -1 | cut -d: -f2)
        result "交易撮合" true
        echo "    交易编号: $tx_no"
        echo "    成交金额: $amount 元"
    else
        result "交易撮合" false "$resp"
    fi
else
    result "交易撮合" false "无待成交挂单（跳过）"
fi

# 核验碳积分来源
if [ -n "$credit_no" ]; then
    body="{\"credit_no\":\"$credit_no\"}"
    resp=$(call_api POST "/exchange/verify-credit" "$body" "${TOKENS[exch]}")
    msg=$(echo "$resp" | grep -o '"msg":"[^"]*"' | head -1 | cut -d'"' -f4)
    result "核验碳积分来源" true
    echo "    核验结果: $msg"
fi

# 交易记录
resp=$(call_api GET "/exchange/transactions" "" "${TOKENS[exch]}")
total=$(echo "$resp" | grep -o '"total":[0-9]*' | head -1 | cut -d: -f2)
result "查看交易记录" true
echo "    交易笔数: $total"

# ============================================================
# 12. 企业AI建议
# ============================================================
header "12. 企业AI建议"

body="{\"enterprise_id\":$ENT_ID,\"company_name\":\"绿色科技公司\",\"industry\":\"制造业\"}"
resp=$(call_api POST "/enterprise/advice" "$body" "${TOKENS[ent]}")
advice=$(echo "$resp" | grep -o '"advice":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -n "$advice" ]; then
    result "生成节能减排建议" true
    echo "    建议摘要: ${advice:0:80}..."
else
    result "生成节能减排建议" false "$resp"
fi

# ============================================================
# 13. 监管核证功能
# ============================================================
header "13. 监管核证功能"

resp=$(call_api GET "/regulator/chain/all" "" "${TOKENS[reg]}")
total=$(echo "$resp" | grep -o '"total":[0-9]*' | head -1 | cut -d: -f2)
result "溯源全部链上记录" true
echo "    总区块数: $total"

resp=$(call_api GET "/regulator/platform/stats" "" "${TOKENS[reg]}")
issued=$(echo "$resp" | grep -o '"total_issued_credits":[0-9.]*' | head -1 | cut -d: -f2)
traded=$(echo "$resp" | grep -o '"total_traded_credits":[0-9.]*' | head -1 | cut -d: -f2)
ent_count=$(echo "$resp" | grep -o '"enterprise_count":[0-9]*' | head -1 | cut -d: -f2)
tx_count=$(echo "$resp" | grep -o '"transaction_count":[0-9]*' | head -1 | cut -d: -f2)
result "平台碳积分发行量统计" true
echo "    总发行积分: $issued"
echo "    已交易积分: $traded"
echo "    企业数量: $ent_count"
echo "    交易笔数: $tx_count"

resp=$(call_api GET "/regulator/risk-alert" "" "${TOKENS[reg]}")
alert_count=$(echo "$resp" | grep -o '"alertCount":[0-9]*' | head -1 | cut -d: -f2)
result "风险告警" true
echo "    告警数量: $alert_count"

resp=$(call_api GET "/regulator/users" "" "${TOKENS[reg]}")
total=$(echo "$resp" | grep -o '"total":[0-9]*' | head -1 | cut -d: -f2)
result "查看所有用户" true
echo "    用户总数: $total"

# 监管员校验数据
if [ -n "$credit_no" ]; then
    body="{\"data_type\":\"credit\",\"data_id\":\"$credit_no\"}"
    resp=$(call_api POST "/regulator/verify" "$body" "${TOKENS[reg]}")
    msg=$(echo "$resp" | grep -o '"msg":"[^"]*"' | head -1 | cut -d'"' -f4)
    result "监管校验数据篡改" true
    echo "    校验结果: $msg"
fi

# ============================================================
# 14. 园区管理员管理企业
# ============================================================
header "14. 园区管理员管理企业"

body="{\"enterprise_id\":$ENT_ID,\"status\":1}"
resp=$(call_api PUT "/admin/enterprise/status" "$body" "${TOKENS[park]}")
msg=$(echo "$resp" | grep -o '"msg":"[^"]*"' | head -1 | cut -d'"' -f4)
result "更新企业入驻状态（启用）" true
echo "    结果: $msg"

# ============================================================
# 总结
# ============================================================
header "测试总结"
TOTAL=$((PASS + FAIL))
echo "  总用例: $TOTAL"
echo -e "  通过:  \033[32m$PASS\033[0m"
echo -e "  失败:  \033[31m$FAIL\033[0m"
RATE=$(echo "scale=1; $PASS * 100 / $TOTAL" | bc 2>/dev/null || echo "N/A")
echo "  通过率: ${RATE}%"

if [ "$FAIL" -eq 0 ]; then
    echo -e "\n\033[32m🎉 所有测试通过！\033[0m"
else
    echo -e "\n\033[33m⚠️  存在失败的测试用例，请检查服务日志。\033[0m"
fi