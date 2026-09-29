// Package database seed 模块 - 全量业务模拟数据初始化
// 所有 seed 函数均为幂等(count→insert)，服务反复重启不会重复插入。
// 依赖 initPresetUsers 已创建预置账号后调用。
package database

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"blockchain-demo/blockchain"
	"blockchain-demo/models"

	"gorm.io/gorm"
)

// SeedAll 执行全量业务模拟数据初始化
func SeedAll() {
	fmt.Println("🌱 开始初始化业务模拟数据...")

	var entUser, parkUser, exchUser, regUser models.User
	DB.Where("username = ?", "小微企业001").First(&entUser)
	DB.Where("username = ?", "园区管理员001").First(&parkUser)
	DB.Where("username = ?", "碳交易所001").First(&exchUser)
	DB.Where("username = ?", "监管核查001").First(&regUser)
	if entUser.ID == 0 {
		fmt.Println("⚠️  未找到预置账号，跳过业务数据初始化")
		return
	}
	_ = parkUser
	_ = regUser

	seedEnergyRecords(entUser.ID)
	seedCarbonCredits(entUser.ID)
	seedSellOrders(entUser.ID)
	seedTransactions(entUser.ID, exchUser.ID)
	seedIoTDevices(entUser.ID)
	seedIoTRecords(entUser.ID)
	seedZKProofs(entUser.ID)
	seedPledgeOrders(entUser.ID)
	seedArbitrations(entUser.ID, exchUser.ID)
	seedCarbonArchives(entUser.ID)
	seedAgentRecords(entUser.ID)
	seedRollupBatches()
	seedProductFootprints(entUser.ID)
	seedOperationLogs(entUser.ID, parkUser.ID, exchUser.ID, regUser.ID)

	// 存量数据补链(幂等)：修复历史上伪哈希数据，保证档案列表"核验链上"可通过
	repairEnterpriseNames()
	// 企业名录去重(幂等)：同名企业仅保留最新注册的一条，旧条目业务数据整体迁移后删除
	repairDedupEnterprise()
	// 演示数据时间线统一平移到 2026年9月以后(幂等)：业务时间+业务单号+链上时间戳整体迁移，
	// 并整链重封存，保证链完整性校验(computeHash==BlockHash)与业务哈希校验(VerifyDataIntegrity)全部通过
	repairShiftTimelineToSeptember()
	repairCreditsOnChain()
	repairArchivesOnChain()
	repairTradesOnChain()
	// 存量挂单绑定修复(幂等)：历史种子挂单无关联积分(credit_id=0)会导致撮合报"关联碳积分不存在"
	bindOrderCredits()
	// 仲裁案件修复(幂等)：历史种子案件重新绑定真实上链交易 + 全量补链
	repairArbitrationsFromTrades()

	fmt.Println("✅ 业务模拟数据初始化完成")
}

// ===== 工具 =====

func fakeHash(seed string) string {
	h := sha256.Sum256([]byte(seed + fmt.Sprintf("%d", time.Now().UnixNano())))
	return "0x" + hex.EncodeToString(h[:])
}

func daysAgo(n int) time.Time { return time.Now().AddDate(0, 0, -n) }

// chainHash 将业务数据真实上链(幂等)并返回链上区块哈希；失败返回空串。
// AddBlockTx 幂等：同一 dataType+dataID 已存在区块时直接返回已有区块，重启不会重复上链。
func chainHash(dataType, dataID string, data interface{}) string {
	blk, err := blockchain.AddBlockTx(DB, dataType, dataID, data)
	if err != nil {
		fmt.Printf("  ⚠️  上链失败 %s/%s: %v\n", dataType, dataID, err)
		return ""
	}
	return blk.BlockHash
}

// repairCreditsOnChain 存量碳积分补链(幂等)：修复历史上由伪哈希生成或漏上链的积分，
// 以真实业务要素重新上链并回写真实区块哈希，保证碳信用档案"核验链上"可通过。
func repairCreditsOnChain() {
	var credits []models.CarbonCredit
	DB.Find(&credits)
	repaired := 0
	for _, cr := range credits {
		blk, err := blockchain.AddBlockTx(DB, models.DataTypeCredit, cr.CreditNo, cr)
		if err != nil {
			fmt.Printf("  ⚠️  碳积分补链失败 %s: %v\n", cr.CreditNo, err)
			continue
		}
		if blk.BlockHash != cr.BlockHash {
			DB.Model(&models.CarbonCredit{}).Where("id = ?", cr.ID).
				Updates(map[string]interface{}{"block_hash": blk.BlockHash, "on_chain": true})
			repaired++
		}
	}
	if repaired > 0 {
		fmt.Printf("  🔧 碳积分补链修复: %d 条\n", repaired)
	}
}

// repairArchivesOnChain 存量碳信用档案补链(幂等)：修复伪哈希档案，回写真实区块哈希。
func repairArchivesOnChain() {
	var archives []models.CarbonArchive
	DB.Find(&archives)
	repaired := 0
	for _, a := range archives {
		blk, err := blockchain.AddBlockTx(DB, models.DataTypeArchive, a.ArchiveNo, a)
		if err != nil {
			fmt.Printf("  ⚠️  档案补链失败 %s: %v\n", a.ArchiveNo, err)
			continue
		}
		if blk.BlockHash != a.BlockHash {
			DB.Model(&models.CarbonArchive{}).Where("id = ?", a.ID).
				Updates(map[string]interface{}{"block_hash": blk.BlockHash, "on_chain": true})
			repaired++
		}
	}
	if repaired > 0 {
		fmt.Printf("  🔧 碳信用档案补链修复: %d 条\n", repaired)
	}
}

// ===== 1. 能耗记录 =====

func seedEnergyRecords(entID uint) {
	var count int64
	DB.Model(&models.EnergyRecord{}).Where("enterprise_id = ?", entID).Count(&count)
	if count >= 8 {
		fmt.Printf("  ℹ️  能耗记录已存在 %d 条，跳过\n", count)
		return
	}
	records := []models.EnergyRecord{
		{RecordNo: "ENERGY-20260901-001", EnterpriseID: entID, DeviceID: "MANUAL", Electricity: 450, Gas: 28, Water: 10, CollectTime: daysAgo(1), BlockHash: fakeHash("e1"), OnChain: true},
		{RecordNo: "ENERGY-20260815-002", EnterpriseID: entID, DeviceID: "MANUAL", Electricity: 520, Gas: 32, Water: 12, CollectTime: daysAgo(15), BlockHash: fakeHash("e2"), OnChain: true},
		{RecordNo: "ENERGY-20260801-003", EnterpriseID: entID, DeviceID: "MANUAL", Electricity: 380, Gas: 22, Water: 8, CollectTime: daysAgo(30), BlockHash: fakeHash("e3"), OnChain: true},
		{RecordNo: "ENERGY-20260715-004", EnterpriseID: entID, DeviceID: "MANUAL", Electricity: 600, Gas: 38, Water: 15, CollectTime: daysAgo(45), BlockHash: fakeHash("e4"), OnChain: true},
		{RecordNo: "ENERGY-20260701-005", EnterpriseID: entID, DeviceID: "MANUAL", Electricity: 420, Gas: 26, Water: 11, CollectTime: daysAgo(60), BlockHash: fakeHash("e5"), OnChain: true},
		{RecordNo: "ENERGY-20260615-006", EnterpriseID: entID, DeviceID: "MANUAL", Electricity: 580, Gas: 35, Water: 14, CollectTime: daysAgo(75), BlockHash: fakeHash("e6"), OnChain: true},
		{RecordNo: "ENERGY-20260601-007", EnterpriseID: entID, DeviceID: "MANUAL", Electricity: 350, Gas: 20, Water: 7, CollectTime: daysAgo(90), BlockHash: fakeHash("e7"), OnChain: true},
		{RecordNo: "ENERGY-20260515-008", EnterpriseID: entID, DeviceID: "MANUAL", Electricity: 480, Gas: 30, Water: 13, CollectTime: daysAgo(105), BlockHash: fakeHash("e8"), OnChain: true},
	}
	for _, r := range records {
		DB.Create(&r)
	}
	fmt.Printf("  ✔  能耗记录: %d 条\n", len(records))
}

// ===== 2. 碳积分 =====

func seedCarbonCredits(entID uint) {
	var count int64
	DB.Model(&models.CarbonCredit{}).Where("enterprise_id = ?", entID).Count(&count)
	if count >= 8 {
		fmt.Printf("  ℹ️  碳积分已存在 %d 条，跳过\n", count)
		return
	}
	credits := []struct {
		no     string
		em, cr float64
		st     string
	}{
		{"CREDIT-20260901-001", 290.0, 290.0, "available"},
		{"CREDIT-20260815-002", 335.0, 335.0, "available"},
		{"CREDIT-20260801-003", 245.0, 245.0, "available"},
		{"CREDIT-20260715-004", 385.0, 385.0, "locked"},
		{"CREDIT-20260701-005", 270.0, 270.0, "sold"},
		{"CREDIT-20260615-006", 370.0, 370.0, "available"},
		{"CREDIT-20260601-007", 220.0, 220.0, "sold"},
		{"CREDIT-20260515-008", 310.0, 310.0, "available"},
	}
	for _, c := range credits {
		cr := models.CarbonCredit{
			CreditNo: c.no, EnterpriseID: entID,
			TotalEmission: c.em, CarbonCredits: c.cr, Status: c.st,
			OwnerID: entID,
		}
		// 真实上链：以 ChainCanonical 签发要素计算业务哈希并追加区块
		cr.BlockHash = chainHash(models.DataTypeCredit, cr.CreditNo, cr)
		cr.OnChain = cr.BlockHash != ""
		DB.Create(&cr)
	}
	fmt.Printf("  ✔  碳积分: %d 条\n", len(credits))
}

// ===== 3. 挂单 =====

func seedSellOrders(entID uint) {
	var count int64
	DB.Model(&models.SellOrder{}).Count(&count)
	if count >= 6 {
		fmt.Printf("  ℹ️  挂单已存在 %d 条，跳过\n", count)
		return
	}
	orders := []models.SellOrder{
		{OrderNo: "ORD-20260901-001", EnterpriseID: entID, Quantity: 200, UnitPrice: 55, TotalAmount: 11000, Status: "pending", OnChain: true, BlockHash: fakeHash("o1"), CreatedAt: daysAgo(1)},
		{OrderNo: "ORD-20260828-002", EnterpriseID: entID, Quantity: 150, UnitPrice: 52, TotalAmount: 7800, Status: "pending", OnChain: true, BlockHash: fakeHash("o2"), CreatedAt: daysAgo(4)},
		{OrderNo: "ORD-20260820-003", EnterpriseID: entID, Quantity: 300, UnitPrice: 48, TotalAmount: 14400, Status: "pending", OnChain: true, BlockHash: fakeHash("o3"), CreatedAt: daysAgo(12)},
		{OrderNo: "ORD-20260810-004", EnterpriseID: entID, Quantity: 100, UnitPrice: 60, TotalAmount: 6000, Status: "pending", OnChain: true, BlockHash: fakeHash("o4"), CreatedAt: daysAgo(22)},
		{OrderNo: "ORD-20260725-005", EnterpriseID: entID, Quantity: 250, UnitPrice: 53, TotalAmount: 13250, Status: "matched", OnChain: true, BlockHash: fakeHash("o5"), CreatedAt: daysAgo(40)},
		{OrderNo: "ORD-20260710-006", EnterpriseID: entID, Quantity: 180, UnitPrice: 50, TotalAmount: 9000, Status: "matched", OnChain: true, BlockHash: fakeHash("o6"), CreatedAt: daysAgo(55)},
	}
	for _, o := range orders {
		DB.Create(&o)
	}
	fmt.Printf("  ✔  挂单: %d 条\n", len(orders))
	// 新种子挂单同样补齐积分绑定，保证撮合流程可用
	bindOrderCredits()
}

// bindOrderCredits 挂单积分绑定修复(幂等)：
// 历史种子挂单未关联碳积分(credit_id=0)，撮合时查不到积分报"关联碳积分不存在"。
// 修复规则：
//   - pending 挂单 → 绑定卖方一条"余额≥挂单量且未被其他挂单占用"的可用积分，并置为 locked；
//   - matched 挂单 → 绑定卖方一条未被占用的已售积分(仅补齐历史关联，不影响业务)。
//
// 仅处理 credit_id=0 的挂单，重复调用无副作用。
func bindOrderCredits() {
	// 清理历史遗留的无主体RWA挂单(早期RWA接口创建：无企业/积分绑定，永远无法撮合)
	var legacyRWA int64
	DB.Model(&models.SellOrder{}).Where("status = ? AND credit_id = 0", "active").Count(&legacyRWA)
	if legacyRWA > 0 {
		DB.Where("status = ? AND credit_id = 0", "active").Delete(&models.SellOrder{})
		fmt.Printf("  🔧 遗留RWA挂单清理: 删除无主体挂单 %d 条\n", legacyRWA)
	}

	var orders []models.SellOrder
	DB.Where("credit_id = 0 OR credit_id IS NULL").Find(&orders)
	if len(orders) == 0 {
		return
	}

	fixed := 0
	for _, o := range orders {
		// 已被其他挂单占用的积分不可重复绑定(杜绝一积分多卖)
		var usedIDs []uint
		DB.Model(&models.SellOrder{}).Where("credit_id > 0").Distinct().Pluck("credit_id", &usedIDs)

		var credit models.CarbonCredit
		q := DB.Where("owner_id = ?", o.EnterpriseID)
		if len(usedIDs) > 0 {
			q = q.Where("id NOT IN ?", usedIDs)
		}
		if o.Status == "matched" {
			// 历史成交单：绑定卖方已售积分
			q = q.Where("status = ?", "sold")
		} else {
			// 在售挂单：绑定余额足够的可用积分并锁定
			q = q.Where("status = ? AND carbon_credits >= ?", "available", o.Quantity)
		}
		if err := q.Order("id asc").First(&credit).Error; err != nil {
			fmt.Printf("  ⚠️  挂单 %s 无可绑定的碳积分(余额不足或已占满)，保持原状\n", o.OrderNo)
			continue
		}

		err := DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.SellOrder{}).Where("id = ?", o.ID).Update("credit_id", credit.ID).Error; err != nil {
				return err
			}
			// 在售挂单绑定的积分同步锁定，与"挂单即锁定"业务规则一致
			if o.Status == "pending" {
				return tx.Model(&models.CarbonCredit{}).
					Where("id = ? AND status = ?", credit.ID, "available").
					Update("status", "locked").Error
			}
			return nil
		})
		if err != nil {
			fmt.Printf("  ⚠️  挂单 %s 绑定积分失败: %v\n", o.OrderNo, err)
			continue
		}
		fixed++
	}
	if fixed > 0 {
		fmt.Printf("  ✔  挂单积分绑定修复: %d 条挂单补齐 credit_id\n", fixed)
	}
}

// ===== 4. 交易 =====

func seedTransactions(entID, exchID uint) {
	var count int64
	DB.Model(&models.Transaction{}).Count(&count)
	if count >= 3 {
		fmt.Printf("  ℹ️  交易已存在 %d 条，跳过\n", count)
		return
	}
	txs := []models.Transaction{
		{TxNo: "TX-20260801-001", OrderNo: "ORD-20260725-005", SellerID: entID, BuyerID: exchID, Quantity: 250, UnitPrice: 53, TotalAmount: 13250, BlockHash: fakeHash("tx1"), OnChain: true, CreatedAt: daysAgo(40)},
		{TxNo: "TX-20260715-002", OrderNo: "ORD-20260710-006", SellerID: entID, BuyerID: exchID, Quantity: 180, UnitPrice: 50, TotalAmount: 9000, BlockHash: fakeHash("tx2"), OnChain: true, CreatedAt: daysAgo(55)},
		{TxNo: "TX-20260601-003", OrderNo: "ORD-20260615-007", SellerID: entID, BuyerID: exchID, Quantity: 220, UnitPrice: 49, TotalAmount: 10780, BlockHash: fakeHash("tx3"), OnChain: true, CreatedAt: daysAgo(100)},
	}
	for _, t := range txs {
		DB.Create(&t)
	}
	fmt.Printf("  ✔  交易: %d 条\n", len(txs))
}

// ===== 5. IoT 设备 =====

func seedIoTDevices(entID uint) {
	var count int64
	DB.Model(&models.IoTDevice{}).Where("enterprise_id = ?", entID).Count(&count)
	if count >= 6 {
		fmt.Printf("  ℹ️  IoT 设备已存在 %d 台，跳过\n", count)
		return
	}
	devices := []models.IoTDevice{
		{DeviceID: "IOT-EM-001", DeviceName: "车间电表 A", EnterpriseID: entID, DeviceType: "meter", Status: "online", LastOnline: time.Now()},
		{DeviceID: "IOT-EM-002", DeviceName: "车间电表 B", EnterpriseID: entID, DeviceType: "meter", Status: "online", LastOnline: time.Now()},
		{DeviceID: "IOT-GM-001", DeviceName: "天然气计量表", EnterpriseID: entID, DeviceType: "meter", Status: "online", LastOnline: time.Now()},
		{DeviceID: "IOT-WM-001", DeviceName: "水表 1 号", EnterpriseID: entID, DeviceType: "meter", Status: "online", LastOnline: time.Now()},
		{DeviceID: "IOT-GW-001", DeviceName: "边缘网关", EnterpriseID: entID, DeviceType: "gateway", Status: "online", LastOnline: time.Now()},
		{DeviceID: "IOT-SE-001", DeviceName: "温度传感器", EnterpriseID: entID, DeviceType: "sensor", Status: "offline", LastOnline: daysAgo(2)},
	}
	for _, d := range devices {
		DB.Create(&d)
	}
	fmt.Printf("  ✔  IoT 设备: %d 台\n", len(devices))
}

// ===== 6. IoT 采集记录 =====

func seedIoTRecords(entID uint) {
	var count int64
	DB.Model(&models.IoTRecord{}).Where("enterprise_id = ?", entID).Count(&count)
	if count >= 12 {
		fmt.Printf("  ℹ️  IoT 采集记录已存在 %d 条，跳过\n", count)
		return
	}
	srcs := []string{"auto", "auto", "auto", "auto", "manual", "manual", "manual", "manual", "auto", "auto", "auto", "auto"}
	risk := []string{"", "", "", "", "人工标记-中风险", "", "人工标记-中风险", "", "", "", "", ""}
	for i := 0; i < 12; i++ {
		DB.Create(&models.IoTRecord{
			RecordNo: fmt.Sprintf("IOT-REC-202609-%03d", 1000+i),
			DeviceID: "IOT-EM-001", EnterpriseID: entID,
			Source: srcs[i], RiskLabel: risk[i],
			Electricity: 80 + rand.Float64()*120,
			Gas:         5 + rand.Float64()*20, Water: 2 + rand.Float64()*8,
			CollectTime: daysAgo(i), BlockHash: fakeHash(fmt.Sprintf("ir%d", i)),
			OnChain: true,
		})
	}
	fmt.Printf("  ✔  IoT 采集记录: 12 条\n")
}

// ===== 7. ZKP 证明 =====

func seedZKProofs(entID uint) {
	var count int64
	DB.Model(&models.ZKProofRecord{}).Where("prover_id = ?", entID).Count(&count)
	if count >= 4 {
		fmt.Printf("  ℹ️  ZKP 证明已存在 %d 条，跳过\n", count)
		return
	}
	proofs := []struct {
		no, ptype, fields, pub, data string
		days                         int
	}{
		{"ZKP-20260905-001", "selective_disclosure", `["carbon_credits","total_emission"]`, `{"carbon_credits":290,"total_emission":290,"period":"***","energy_breakdown":"***"}`, "模拟Groth16 JSON证明体", 4},
		{"ZKP-20260820-002", "selective_disclosure", `["carbon_credits"]`, `{"carbon_credits":335}`, "模拟Groth16 JSON证明体", 20},
		{"ZKP-20260730-003", "selective_disclosure", `["carbon_credits","total_emission","period"]`, `{"carbon_credits":385,"total_emission":385}`, "模拟Groth16 JSON证明体", 40},
		{"ZKP-20260710-004", "pqc", `["total_emission"]`, `{"total_emission":270}`, "CRYSTALS-Dilithium PQC签名", 60},
	}
	for _, p := range proofs {
		DB.Create(&models.ZKProofRecord{
			ProofNo: p.no, ProofType: p.ptype, ProverID: entID,
			FieldSelector: p.fields, PublicData: p.pub, ProofData: p.data,
			BlockHash: fakeHash("zkp" + p.no), OnChain: true, CreatedAt: daysAgo(p.days),
		})
	}
	fmt.Printf("  ✔  ZKP 证明: %d 条\n", len(proofs))
}

// ===== 8. 质押订单 =====

func seedPledgeOrders(entID uint) {
	var count int64
	DB.Model(&models.PledgeOrder{}).Where("enterprise_id = ?", entID).Count(&count)
	if count == 0 {
		// 已还清订单的还款金额 = 本金 + 利息(4000 × (1 + 0.06 × 3/12) = 4060)
		pledges := []models.PledgeOrder{
			{PledgeNo: "PLEDGE-20260820-001", EnterpriseID: entID, PledgeAmount: 200, LoanAmount: 8000, TermMonths: 6, AnnualRate: 0.07, Status: "active", BlockHash: fakeHash("pl1"), OnChain: true, CreatedAt: daysAgo(20)},
			{PledgeNo: "PLEDGE-20260701-002", EnterpriseID: entID, PledgeAmount: 150, LoanAmount: 6000, TermMonths: 12, AnnualRate: 0.08, Status: "active", BlockHash: fakeHash("pl2"), OnChain: true, CreatedAt: daysAgo(70)},
			{PledgeNo: "PLEDGE-20260515-003", EnterpriseID: entID, PledgeAmount: 100, LoanAmount: 4000, TermMonths: 3, AnnualRate: 0.06, RepaidAmount: 4060, Status: "cleared", BlockHash: fakeHash("pl3"), OnChain: true, CreatedAt: daysAgo(120)},
		}
		for _, p := range pledges {
			DB.Create(&p)
		}
		fmt.Printf("  ✔  质押订单: %d 条\n", len(pledges))
	}

	// 存量数据修复：老版本订单缺失期限/利率时按默认业务规则回填(6个月/年化7%)，
	// 已还清但未记录还款金额的订单按 本金×(1+年利率×期限/12) 补记，保证页面统计一致
	DB.Model(&models.PledgeOrder{}).
		Where("term_months IS NULL OR term_months = 0").
		Updates(map[string]interface{}{"term_months": 6, "annual_rate": 0.07})
	DB.Model(&models.PledgeOrder{}).
		Where("status = ? AND (repaid_amount IS NULL OR repaid_amount = 0)", "cleared").
		Update("repaid_amount", gorm.Expr("ROUND(loan_amount * (1 + COALESCE(annual_rate, 0.07) * COALESCE(term_months, 6) / 12), 2)"))
}

// ===== 9. 仲裁案件 =====

// arbDescFromTrade 由真实链上交易生成案件描述(企业名称/数量/金额全部取自交易数据，不编造)
func arbDescFromTrade(t models.Transaction, sellerName, buyerName string) string {
	return fmt.Sprintf(
		"链上交易 %s：卖方 %s 向买方 %s 转让 %.0f kgCO₂ 碳积分，成交单价 %.0f 元/积分，成交金额 %.2f 元。该交易已产生纠纷，现提起交易仲裁。",
		t.TxNo, sellerName, buyerName, t.Quantity, t.UnitPrice, t.TotalAmount,
	)
}

// userNameByID 取企业展示名称(优先企业名称，回退账号)
func userNameByID(id uint) string {
	var u models.User
	if err := DB.Select("company", "username").First(&u, id).Error; err != nil {
		return "未知企业"
	}
	if u.Company != "" {
		return u.Company
	}
	return u.Username
}

// seedArbitrations 仲裁案件种子(真实数据)：
// 案件不再凭空编造——仅当一笔已上链交易产生纠纷时才生成对应案件，
// 原告/被告/关联交易/案件描述全部取自链上交易记录。
func seedArbitrations(entID, exchID uint) {
	var count int64
	DB.Model(&models.ArbitrationCase{}).Count(&count)
	if count >= 3 {
		fmt.Printf("  ℹ️  仲裁案件已存在 %d 条，跳过\n", count)
		return
	}
	// 只从已上链交易生成案件
	var trades []models.Transaction
	DB.Where("on_chain = ?", true).Order("id asc").Limit(3).Find(&trades)
	if len(trades) == 0 {
		fmt.Println("  ℹ️  暂无已上链交易，跳过仲裁案件生成")
		return
	}
	statuses := []string{"pending", "under_review", "resolved"}
	created := 0
	for i, t := range trades {
		caseNo := fmt.Sprintf("ARB-%s-%03d", t.CreatedAt.Format("20060102"), i+1)
		arb := models.ArbitrationCase{
			CaseNo:        caseNo,
			ApplicantID:   t.SellerID, // 原告=交易卖方
			RespondentID:  t.BuyerID,  // 被告=交易买方
			TransactionNo: t.TxNo,     // 关联真实上链交易
			DisputeAmount: t.TotalAmount,
			CaseType:      "credit_dispute",
			Description:   arbDescFromTrade(t, userNameByID(t.SellerID), userNameByID(t.BuyerID)),
			Status:        statuses[i%len(statuses)],
		}
		if arb.Status == "resolved" {
			arb.Verdict = "经链上核验交易凭证与积分权属转移记录一致，卖方胜诉，买方限期支付尾款"
			arb.ResolvedBy = exchID
			now := time.Now()
			arb.ResolvedAt = &now
		}
		if err := DB.Create(&arb).Error; err != nil {
			continue
		}
		// 案件真实上链存证
		if blk, err := blockchain.AddBlockTx(DB, models.DataTypeArbitration, arb.CaseNo, arb); err == nil {
			DB.Model(&models.ArbitrationCase{}).Where("id = ?", arb.ID).
				Updates(map[string]interface{}{"block_hash": blk.BlockHash, "on_chain": true})
		}
		created++
	}
	fmt.Printf("  ✔  仲裁案件(源自已上链交易): %d 条\n", created)
}

// repairEnterpriseNames 企业名称统一(幂等)：将平台企业名称规范为全称，
// 企业名录、碳信用档案、仲裁案件等所有展示处自动同步。
func repairEnterpriseNames() {
	nameMapping := []struct {
		prefix string // 按账号/企业名前缀匹配
		full   string // 规范后的企业全称
	}{
		{"小微企业001", "绿恒节能科技有限公司"},
		{"晨光", "晨光烘焙食品有限公司"},
		{"恒达", "恒达针织纺织品有限公司"},
		{"精工", "精工不锈钢制品有限公司"},
		{"蓝天", "蓝天包装制品有限公司"},
	}
	var users []models.User
	DB.Where("role = ?", models.RoleEnterprise).Find(&users)
	changed := 0
	for _, u := range users {
		target := ""
		for _, m := range nameMapping {
			if strings.HasPrefix(u.Username, m.prefix) || strings.HasPrefix(u.Company, m.prefix) {
				target = m.full
				break
			}
		}
		if target == "" || u.Company == target {
			continue
		}
		DB.Model(&models.User{}).Where("id = ?", u.ID).Update("company", target)
		changed++
	}
	if changed > 0 {
		fmt.Printf("  🔧 企业名称规范更新: %d 家\n", changed)
	}
}

// repairDedupEnterprise 企业名录去重(幂等)：同名企业仅保留最新注册的一条，
// 旧条目名下业务数据(积分/挂单/交易/能耗/IoT/档案/足迹等)整体迁移到保留条目后删除旧条目。
func repairDedupEnterprise() {
	type entRow struct {
		ID        uint
		Company   string
		CreatedAt time.Time
	}
	var list []entRow
	DB.Model(&models.User{}).
		Select("id, company, created_at").
		Where("role = ? AND company != ''", models.RoleEnterprise).
		Order("created_at asc").
		Scan(&list)

	// 按企业全称分组
	groups := map[string][]entRow{}
	for _, e := range list {
		groups[e.Company] = append(groups[e.Company], e)
	}
	for _, es := range groups {
		if len(es) < 2 {
			continue
		}
		keep := es[len(es)-1] // 保留最新注册的一条
		for _, old := range es[:len(es)-1] {
			migrateUserRefs(old.ID, keep.ID)
			DB.Where("id = ?", old.ID).Delete(&models.User{})
			fmt.Printf("  🔧 企业名录去重: %s (#%d @%s) 已合并至 #(%d @%s)\n",
				old.Company, old.ID, old.CreatedAt.Format("2006-01-02 15:04"), keep.ID, keep.CreatedAt.Format("2006-01-02 15:04"))
		}
	}
}

// migrateUserRefs 将旧用户名下全部业务引用迁移到新用户(表/列不存在时自动忽略)
func migrateUserRefs(oldID, newID uint) {
	refs := []struct{ table, col string }{
		{"carbon_credits", "enterprise_id"}, {"carbon_credits", "owner_id"},
		{"sell_orders", "enterprise_id"}, {"sell_orders", "buyer_id"},
		{"transactions", "seller_id"}, {"transactions", "buyer_id"},
		{"arbitration_cases", "applicant_id"}, {"arbitration_cases", "respondent_id"},
		{"energy_records", "enterprise_id"},
		{"io_t_devices", "enterprise_id"}, {"io_t_records", "enterprise_id"},
		{"pledge_orders", "enterprise_id"},
		{"carbon_archives", "enterprise_id"},
		{"product_footprints", "enterprise_id"},
	}
	for _, r := range refs {
		DB.Exec(fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s = ?", r.table, r.col, r.col), newID, oldID)
	}
}

// ==================== 演示数据时间线平移(2026-09 以后) ====================

// 时间平移基准：2026-09-01 08:00 起，按"原时间降序"逐行 +37 分钟排布，
// 既保证全部落在 9 月以后，又完整保留各表内部的时间先后顺序。
var timelineBase = time.Date(2026, 9, 1, 8, 0, 0, 0, time.Local)

// repairShiftTimelineToSeptember 把 2026-09-01 之前的演示数据时间整体平移到 9 月时间线：
//  1. 业务表时间列(上报/采集/核算/挂单/成交/存证/质押/证明/足迹/报告/日志/仲裁/批量)；
//  2. 业务单号中的日期段(202605~202608 → 202609，含链上 block_records.data_id，保证溯源可达)；
//  3. 整链重封存：区块时间戳平移 + 漂移业务哈希重算 + 链式重算哈希 + 回写业务行。
//     用户注册时间(users.created_at)不在平移范围(企业名录展示的注册时间均为 9 月)。
func repairShiftTimelineToSeptember() {
	// ---- 1. 业务时间列平移(< 2026-09-01 才处理，天然幂等) ----
	timeCols := []struct{ table, col string }{
		{"energy_records", "collect_time"}, {"energy_records", "created_at"},
		{"io_t_records", "collect_time"}, {"io_t_records", "created_at"},
		{"carbon_credits", "created_at"},
		{"sell_orders", "created_at"},
		{"transactions", "created_at"},
		{"carbon_archives", "created_at"},
		{"pledge_orders", "created_at"},
		{"zk_proof_records", "created_at"},
		{"product_footprints", "created_at"},
		{"report_records", "created_at"},
		{"operation_logs", "created_at"},
		{"agent_records", "created_at"},
		{"rollup_batches", "created_at"}, {"rollup_batches", "verified_at"},
		{"arbitration_cases", "created_at"}, {"arbitration_cases", "resolved_at"},
		{"block_records", "created_at"},
	}
	shifted := 0
	for _, tc := range timeCols {
		if shiftTableTimes(tc.table, tc.col) {
			shifted++
		}
	}

	// ---- 2. 业务单号日期段替换(202605~202608 → 202609，保持唯一性与引用一致性) ----
	noCols := []struct{ table, col string }{
		{"energy_records", "record_no"}, {"carbon_credits", "credit_no"},
		{"sell_orders", "order_no"},
		{"transactions", "tx_no"}, {"transactions", "order_no"},
		{"carbon_archives", "archive_no"}, {"carbon_archives", "credit_no"},
		{"pledge_orders", "pledge_no"}, {"zk_proof_records", "proof_no"},
		{"product_footprints", "product_no"}, {"report_records", "report_no"},
		{"agent_records", "agent_no"}, {"rollup_batches", "batch_no"},
		{"arbitration_cases", "case_no"}, {"arbitration_cases", "transaction_no"},
		{"block_records", "data_id"}, {"io_t_records", "record_no"},
		{"operation_logs", "data_id"},
	}
	renamed := 0
	for _, nc := range noCols {
		if replaceNoDateSegment(nc.table, nc.col) {
			renamed++
		}
	}

	// ---- 3. 整链重封存(时间戳平移 + 业务哈希重算 + 链式哈希重算 + 业务行回写) ----
	resealed := blockchain.Chain.ResealChain(timelineBase.Unix())

	if shifted > 0 || renamed > 0 || resealed > 0 {
		fmt.Printf("  🔧 时间线平移(→2026-09): 时间列 %d 组, 单号列 %d 组, 重封存区块 %d 个\n", shifted, renamed, resealed)
	}
}

// shiftTableTimes 平移单表单列：返回是否有行被平移。
// 原时间越新的行平移后越新(按原时间降序 +37min/行)，表内时间先后关系不变。
func shiftTableTimes(table, col string) bool {
	cutoff := timelineBase.AddDate(0, 0, -1) // 2026-08-31 08:00 之前的统一平移
	var rows []struct {
		ID uint
	}
	if err := DB.Table(table).
		Select("id").
		Where(col+" IS NOT NULL AND "+col+" < ?", cutoff).
		Order(col + " desc").
		Scan(&rows).Error; err != nil || len(rows) == 0 {
		return false
	}
	for i, r := range rows {
		nt := timelineBase.Add(time.Duration(i) * 37 * time.Minute)
		if err := DB.Table(table).Where("id = ?", r.ID).Update(col, nt).Error; err != nil {
			return false
		}
	}
	return true
}

// replaceNoDateSegment 替换业务单号中的旧日期段(202605~202608 → 202609)，返回是否有更新。
// 各类型单号尾部序号唯一，月份归并后不会产生冲突；链上 block_records.data_id 同步替换保证溯源可达。
func replaceNoDateSegment(table, col string) bool {
	sql := fmt.Sprintf(
		"UPDATE %s SET %s = replace(replace(replace(replace(%s,'202605','202609'),'202606','202609'),'202607','202609'),'202608','202609') "+
			"WHERE %s LIKE '%%202605%%' OR %s LIKE '%%202606%%' OR %s LIKE '%%202607%%' OR %s LIKE '%%202608%%'",
		table, col, col, col, col, col, col)
	res := DB.Exec(sql)
	return res.Error == nil && res.RowsAffected > 0
}

// repairTradesOnChain 存量交易凭证补链(幂等)：修复伪哈希交易，回写真实区块哈希。
func repairTradesOnChain() {
	var txs []models.Transaction
	DB.Find(&txs)
	repaired := 0
	for _, t := range txs {
		blk, err := blockchain.AddBlockTx(DB, models.DataTypeTransaction, t.TxNo, t)
		if err != nil {
			fmt.Printf("  ⚠️  交易补链失败 %s: %v\n", t.TxNo, err)
			continue
		}
		if blk.BlockHash != t.BlockHash {
			DB.Model(&models.Transaction{}).Where("id = ?", t.ID).
				Updates(map[string]interface{}{"block_hash": blk.BlockHash, "on_chain": true})
			repaired++
		}
	}
	if repaired > 0 {
		fmt.Printf("  🔧 交易凭证补链修复: %d 条\n", repaired)
	}
}

// repairArbitrationsFromTrades 存量仲裁案件修复(幂等)：
//  1. 历史种子案件无关联交易(transaction_no为空) → 重新绑定到真实已上链交易；
//     无可用上链交易时按规则删除该案件(无上链交易不生成仲裁案件)；
//  2. 全量案件补链，回写真实区块哈希(修复伪哈希存证)。
func repairArbitrationsFromTrades() {
	var trades []models.Transaction
	DB.Where("on_chain = ?", true).Order("id asc").Find(&trades)

	// 1. 重新绑定历史无交易关联的案件
	var legacy []models.ArbitrationCase
	DB.Where("transaction_no = '' OR transaction_no IS NULL").Find(&legacy)
	rebound, removed := 0, 0
	for i, c := range legacy {
		if i < len(trades) {
			t := trades[i]
			DB.Model(&models.ArbitrationCase{}).Where("id = ?", c.ID).Updates(map[string]interface{}{
				"transaction_no": t.TxNo,
				"applicant_id":   t.SellerID,
				"respondent_id":  t.BuyerID,
				"case_type":      "credit_dispute",
				"dispute_amount": t.TotalAmount,
				"description":    arbDescFromTrade(t, userNameByID(t.SellerID), userNameByID(t.BuyerID)),
				"evidence":       "",
			})
			rebound++
		} else {
			DB.Where("id = ?", c.ID).Delete(&models.ArbitrationCase{})
			removed++
		}
	}

	// 2. 全量补链
	var cases []models.ArbitrationCase
	DB.Find(&cases)
	repaired := 0
	for _, a := range cases {
		blk, err := blockchain.AddBlockTx(DB, models.DataTypeArbitration, a.CaseNo, a)
		if err != nil {
			fmt.Printf("  ⚠️  仲裁补链失败 %s: %v\n", a.CaseNo, err)
			continue
		}
		if blk.BlockHash != a.BlockHash {
			DB.Model(&models.ArbitrationCase{}).Where("id = ?", a.ID).
				Updates(map[string]interface{}{"block_hash": blk.BlockHash, "on_chain": true})
			repaired++
		}
	}
	if rebound > 0 || removed > 0 || repaired > 0 {
		fmt.Printf("  🔧 仲裁案件修复: 重绑 %d 条 / 删除无交易案件 %d 条 / 补链 %d 条\n", rebound, removed, repaired)
	}
}

// ===== 10. 碳信用档案 =====

func seedCarbonArchives(entID uint) {
	var count int64
	DB.Model(&models.CarbonArchive{}).Count(&count)
	if count >= 4 {
		fmt.Printf("  ℹ️  碳信用档案已存在 %d 条，跳过\n", count)
		return
	}
	archives := []struct {
		no, cn, cs, vb string
		total, cr      float64
		days           int
	}{
		{"ARCH-202609-001", "CREDIT-20260901-001", "企业自主申报", "省生态环境厅", 290, 290, 5},
		{"ARCH-202608-002", "CREDIT-20260815-002", "第三方核查机构", "中环联合认证中心", 335, 335, 35},
		{"ARCH-202607-003", "CREDIT-20260715-004", "AI 自动链上生成", "零碳微证智能合约", 385, 385, 65},
		{"ARCH-202606-004", "CREDIT-20260615-006", "企业自主申报", "省生态环境厅", 370, 370, 95},
	}
	for _, a := range archives {
		arc := models.CarbonArchive{
			ArchiveNo: a.no, CreditNo: a.cn, EnterpriseID: entID,
			TotalEmission: a.total, CarbonCredits: a.cr,
			SourceDesc: a.cs, VerifiedBy: a.vb,
			CreatedAt: daysAgo(a.days),
		}
		// 真实上链：与 CreateArchive 接口相同的存证链路
		arc.BlockHash = chainHash(models.DataTypeArchive, arc.ArchiveNo, arc)
		arc.OnChain = arc.BlockHash != ""
		DB.Create(&arc)
	}
	fmt.Printf("  ✔  碳信用档案: %d 条\n", len(archives))
}

// ===== 11. Agent 记录 =====

func seedAgentRecords(entID uint) {
	var count int64
	DB.Model(&models.AgentRecord{}).Count(&count)
	if count >= 5 {
		fmt.Printf("  ℹ️  Agent 记录已存在 %d 条，跳过\n", count)
		return
	}
	agents := []struct {
		no, atype, inp, out, zk string
		days                    int
	}{
		{"AGT-20260905-001", "park_dispatch", "{\"park_id\":1,\"date\":\"2026-09-05\"}", "已下发减排指令至 3 家企业，预计减排 850 kgCO₂", "", 4},
		{"AGT-20260901-002", "risk_detect", "{\"scan_scope\":\"park\"}", "检测到 2 条高风险异常（手动录入标记）", "ZK-RISK-001", 8},
		{"AGT-20260820-003", "trade_agent", "{\"buyer_id\":1,\"max_price\":55}", "成功撮合 3 笔交易，总金额 ¥18,500", "ZK-TRADE-001", 20},
		{"AGT-20260810-004", "risk_detect", "{\"scan_scope\":\"enterprise\"}", "检测到 1 条中风险异常", "", 30},
		{"AGT-20260720-005", "park_dispatch", "{\"park_id\":1}", "已完成园区月度资源调度报告", "", 50},
	}
	for _, a := range agents {
		DB.Create(&models.AgentRecord{
			AgentNo: a.no, AgentType: a.atype, TriggerBy: entID,
			InputData: a.inp, OutputData: a.out, ZKProofRef: a.zk,
			BlockHash: fakeHash("agt" + a.no), OnChain: true, CreatedAt: daysAgo(a.days),
		})
	}
	fmt.Printf("  ✔  Agent 记录: %d 条\n", len(agents))
}

// ===== 12. Rollup 批次 =====

func seedRollupBatches() {
	var count int64
	DB.Model(&models.RollupBatch{}).Count(&count)
	if count >= 3 {
		fmt.Printf("  ℹ️  Rollup 批次已存在 %d 条，跳过\n", count)
		return
	}
	now := time.Now()
	vb1, vb2 := now.AddDate(0, 0, -20), now.AddDate(0, 0, -40)
	batches := []models.RollupBatch{
		{BatchNo: "RB-20260815-001", FromBlock: 18500200, ToBlock: 18501200, TxCount: 420, StateRoot: fakeHash("sr1"), FaultProof: "SHA-256 聚合根一致性校验通过", Status: "verified", VerifiedBy: "监管核查001", CreatedAt: daysAgo(20), VerifiedAt: &vb1},
		{BatchNo: "RB-20260801-002", FromBlock: 18501000, ToBlock: 18502000, TxCount: 315, StateRoot: fakeHash("sr2"), FaultProof: "", Status: "pending", VerifiedBy: "", CreatedAt: daysAgo(40), VerifiedAt: &vb2},
	}
	for _, b := range batches {
		DB.Create(&b)
	}
	fmt.Printf("  ✔  Rollup 批次: +%d 条 (当前 %d)\n", len(batches), count+int64(len(batches)))
}

// ===== 13. 产品碳足迹 =====

func seedProductFootprints(entID uint) {
	var count int64
	DB.Model(&models.ProductFootprint{}).Where("enterprise_id = ?", entID).Count(&count)
	if count >= 8 {
		fmt.Printf("  ℹ️  产品碳足迹已存在 %d 条，跳过\n", count)
		return
	}
	items := []struct {
		no, name, cat, vb                 string
		raw, mfg, tr, usage, waste, total float64
		days                              int
	}{
		{"PFP-202609-001", "节能芯片", "电子", "中环联合认证中心", 85.2, 142.5, 28.3, 260.0, 15.6, 531.6, 5},
		{"PFP-202608-002", "环保建材板", "建材", "省建筑科学研究院", 120.0, 65.5, 45.2, 30.0, 22.0, 282.7, 30},
		{"PFP-202607-003", "低碳包装", "包装", "", 45.0, 38.0, 22.0, 10.0, 40.0, 155.0, 55},
		{"PFP-202606-004", "纺织面料", "纺织", "GOTS 认证机构", 62.5, 95.0, 35.0, 280.0, 18.0, 490.5, 80},
		{"PFP-202605-005", "光伏组件", "电子", "TÜV 莱茵", 180.0, 120.0, 55.0, 45.0, 12.0, 412.0, 105},
		{"PFP-202604-006", "农业机械", "机械", "", 150.0, 200.0, 80.0, 350.0, 25.0, 805.0, 130},
		{"PFP-202603-007", "LED 灯泡", "电子", "", 35.0, 55.0, 18.0, 200.0, 14.0, 322.0, 160},
		{"PFP-202602-008", "饮用瓶装水", "食品", "", 15.0, 25.0, 30.0, 12.0, 35.0, 117.0, 190},
	}
	for _, p := range items {
		DB.Create(&models.ProductFootprint{
			ProductNo: p.no, Name: p.name, Category: p.cat, EnterpriseID: entID,
			RawMaterials: p.raw, Manufacture: p.mfg, Transport: p.tr,
			Usage: p.usage, Waste: p.waste, Total: p.total,
			VerifiedBy: p.vb,
			BlockHash:  fakeHash("pf" + p.no), OnChain: true, CreatedAt: daysAgo(p.days),
		})
	}
	fmt.Printf("  ✔  产品碳足迹: %d 条\n", len(items))
}

// ===== 14. 审计日志 =====

func seedOperationLogs(entID, parkID, exchID, regID uint) {
	var count int64
	DB.Model(&models.OperationLog{}).Count(&count)
	if count >= 10 {
		fmt.Printf("  ℹ️  审计日志已存在 %d 条，跳过\n", count)
		return
	}
	type log struct {
		op, name, role, dt, did, h, detail string
		days                               int
		uid                                uint
	}
	logs := []log{
		{"energy_create", "小微企业001", "enterprise", "energy", "ENERGY-20260901-001", fakeHash("l1"), "手动录入能耗数据: 450kWh电, 28m³天然气, 10t水", 1, entID},
		{"credit_calculate", "小微企业001", "enterprise", "credit", "CREDIT-20260901-001", fakeHash("l2"), "核算碳积分 290 kgCO₂ → 290 积分", 1, entID},
		{"sell_order_create", "小微企业001", "enterprise", "sell_order", "ORD-20260901-001", fakeHash("l3"), "创建卖出挂单: 200 积分 @ ¥55", 1, entID},
		{"trade_match", "碳交易所001", "exchange", "transaction", "TX-20260801-001", fakeHash("l4"), "交易撮合: 250 积分 @ ¥53 → ¥13,250", 40, exchID},
		{"report_generate", "园区管理员001", "park_admin", "report", "REPORT-20260905-001", fakeHash("l5"), "AI 生成园区低碳发展报告", 4, parkID},
		{"on_chain", "监管核查001", "regulator", "chain", "BLOCK-18501200", fakeHash("l6"), "手动补链: 修复 5 条漏上链的交易记录", 15, regID},
		{"credit_transfer", "碳交易所001", "exchange", "transaction", "TX-20260715-002", fakeHash("l7"), "碳积分权属变更: 180 积分 → ¥9,000", 55, exchID},
		{"user_status_update", "监管核查001", "regulator", "user", "小微企业001", fakeHash("l8"), "启用用户账号", 30, regID},
		{"report_generate", "小微企业001", "enterprise", "report", "REPORT-20260909-001", fakeHash("l9"), "AI 生成减排建议报告", 8, entID},
		{"credit_transfer", "碳交易所001", "exchange", "transaction", "TX-20260601-003", fakeHash("l10"), "碳积分权属变更: 220 积分 → ¥10,780", 100, exchID},
	}
	for _, l := range logs {
		DB.Create(&models.OperationLog{
			Operation: l.op, OperatorID: l.uid, OperatorName: l.name, Role: l.role,
			DataType: l.dt, DataID: l.did, BlockHash: l.h, Detail: l.detail,
			IP: "127.0.0.1", CreatedAt: daysAgo(l.days),
		})
	}
	fmt.Printf("  ✔  审计日志: %d 条\n", len(logs))
}
