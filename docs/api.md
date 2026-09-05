# 微碳链 API 接口文档

## 基础信息

- **基础URL**: `http://localhost:8080/api`
- **认证方式**: Bearer Token (JWT)
- **Content-Type**: `application/json`

## 目录

1. [公开接口](#1-公开接口)
2. [认证接口](#2-认证接口)
3. [区块链接口](#3-区块链接口)
4. [企业能耗接口](#4-企业能耗接口)
5. [碳积分接口](#5-碳积分接口)
6. [园区管理员接口](#6-园区管理员接口)
7. [交易所接口](#7-交易所接口)
8. [监管核证接口](#8-监管核证接口)

---

## 1. 公开接口

### 1.1 健康检查

```
GET /api/health
```

**响应示例:**
```json
{
  "status": "ok",
  "service": "微碳链 - 区块链碳积分可信交易平台"
}
```

---

## 2. 认证接口

### 2.1 用户注册

```
POST /api/auth/register
```

**请求体:**
```json
{
  "username": "enterprise01",
  "password": "pass123",
  "company": "绿色科技公司",
  "role": "enterprise",
  "park_id": 1
}
```

**角色说明:**
- `enterprise` - 小微企业企业角色
- `park_admin` - 工业园区管理员角色
- `exchange` - 碳交易所交易角色
- `regulator` - 监管核证角色

### 2.2 用户登录

```
POST /api/auth/login
```

**请求体:**
```json
{
  "username": "enterprise01",
  "password": "pass123"
}
```

**响应:**
```json
{
  "code": 200,
  "msg": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": {
      "id": 1,
      "username": "enterprise01",
      "role": "enterprise",
      "company": "绿色科技公司",
      "park_id": 1,
      "status": 1
    }
  }
}
```

### 2.3 获取当前用户信息

```
GET /api/auth/me
```

**Headers:** `Authorization: Bearer <token>`

---

## 3. 区块链接口

### 3.1 上链存证

```
POST /api/chain/upload
```

**请求体:**
```json
{
  "data_type": "energy",
  "data_id": "ENERGY-20240101-001"
}
```

**data_type 可选值:** `energy` / `credit` / `transaction`

### 3.2 溯源查询

```
GET /api/chain/query?data_no=ENERGY-20240101-001
```

### 3.3 校验数据篡改

```
POST /api/chain/verify
```

**请求体:**
```json
{
  "data_type": "credit",
  "data_id": "CREDIT-20240101-001"
}
```

### 3.4 区块链信息

```
GET /api/chain/info
```

### 3.5 区块列表

```
GET /api/chain/blocks?page=1&page_size=20
```

---

## 4. 企业能耗接口

> 需要角色: enterprise / park_admin / regulator

### 4.1 模拟生成能耗数据

```
POST /api/energy/simulate
```

**请求体:**
```json
{
  "enterprise_id": 1,
  "device_id": "METER-001",
  "count": 5
}
```

### 4.2 查询能耗记录

```
GET /api/energy/list?enterprise_id=1&page=1&page_size=20
```

### 4.3 生成节能减排建议

```
POST /api/enterprise/advice
```

**请求体:**
```json
{
  "enterprise_id": 1,
  "company_name": "绿色科技公司",
  "industry": "制造业"
}
```

---

## 5. 碳积分接口

> 需要角色: enterprise / exchange / park_admin / regulator

### 5.1 核算碳积分并上链

```
POST /api/carbon/calculate
```

**请求体:**
```json
{
  "energy_record_id": 1
}
```

### 5.2 查看碳积分列表

```
GET /api/carbon/my-credits?enterprise_id=1&page=1&page_size=20
```

### 5.3 碳数据统计

```
GET /api/carbon/stats?enterprise_id=1
```

### 5.4 创建挂单卖出

```
POST /api/carbon/sell
```

**请求体:**
```json
{
  "credit_id": 1,
  "quantity": 100,
  "unit_price": 10.5
}
```

### 5.5 查看挂单列表

```
GET /api/carbon/sell-orders?status=pending&page=1&page_size=20
```

---

## 6. 园区管理员接口

> 需要角色: park_admin / regulator

### 6.1 园区碳数据总览

```
GET /api/admin/park/overview?park_id=1
```

### 6.2 园区企业列表

```
GET /api/admin/park/enterprises?park_id=1&page=1&page_size=20
```

### 6.3 管理企业入驻状态

```
PUT /api/admin/enterprise/status
```

**请求体:**
```json
{
  "enterprise_id": 1,
  "status": 1
}
```

### 6.4 生成园区低碳报告

```
POST /api/admin/park/report
```

**请求体:**
```json
{
  "park_id": 1,
  "park_name": "绿色工业园区"
}
```

### 6.5 企业碳数据详情

```
GET /api/admin/enterprise/detail?enterprise_id=1
```

---

## 7. 交易所接口

> 需要角色: exchange / regulator

### 7.1 浏览卖方挂单

```
GET /api/exchange/orders?page=1&page_size=20
```

### 7.2 交易撮合

```
POST /api/exchange/match
```

**请求体:**
```json
{
  "order_id": 1,
  "buyer_id": 2
}
```

### 7.3 核验碳积分来源

```
POST /api/exchange/verify-credit
```

**请求体:**
```json
{
  "credit_no": "CREDIT-20240101-001"
}
```

### 7.4 交易记录

```
GET /api/exchange/transactions?page=1&page_size=20
```

---

## 8. 监管核证接口

> 需要角色: regulator

### 8.1 溯源全部链上记录

```
GET /api/regulator/chain/all?page=1&page_size=20
```

### 8.2 校验任意数据篡改

```
POST /api/regulator/verify
```

**请求体:**
```json
{
  "data_type": "energy",
  "data_id": "ENERGY-20240101-001"
}
```

### 8.3 平台碳积分统计

```
GET /api/regulator/platform/stats
```

### 8.4 风险告警

```
GET /api/regulator/risk-alert
```

### 8.5 查看所有用户

```
GET /api/regulator/users?page=1&page_size=20
```

---

## 快速测试流程

### 1. 注册各角色账号

```bash
# 企业
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"ent1","password":"123","company":"绿色科技","role":"enterprise","park_id":1}'

# 园区管理员
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"park1","password":"123","company":"工业园区管委会","role":"park_admin"}'

# 交易所
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"exch1","password":"123","company":"碳交易所","role":"exchange"}'

# 监管
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"reg1","password":"123","company":"监管中心","role":"regulator"}'
```

### 2. 登录获取Token

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"ent1","password":"123"}'
```

### 3. 模拟能耗数据

```bash
curl -X POST http://localhost:8080/api/energy/simulate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"enterprise_id":2,"count":3}'
```

### 4. 核算碳积分并上链

```bash
curl -X POST http://localhost:8080/api/carbon/calculate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"energy_record_id":1}'
```

### 5. 查询区块链信息

```bash
curl http://localhost:8080/api/chain/info \
  -H "Authorization: Bearer <token>"
```