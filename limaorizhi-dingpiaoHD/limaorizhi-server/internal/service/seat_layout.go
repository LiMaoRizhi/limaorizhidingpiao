package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"gorm.io/gorm"
)

// SeatCell 座位布局中的单个格子
// Type: "seat"普通座位 / "aisle"过道 / "driver"驾驶座 / "empty"空白
type SeatCell struct {
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Type   string `json:"type"`
	SeatNo int    `json:"seat_no,omitempty"`
}

// SeatLayout 车辆座位布局
type SeatLayout struct {
	Rows int        `json:"rows"`
	Cols int        `json:"cols"`
	Cells []SeatCell `json:"cells"`
}

// SeatStatus 座位在当前区间的占用状态
type SeatStatus struct {
	SeatNo    int    `json:"seat_no"`
	Row       int    `json:"row"`
	Col       int    `json:"col"`
	Occupied  bool   `json:"occupied"`  // 当前区间是否被占
	OccupiedBy string `json:"occupied_by,omitempty"` // 占用人(脱敏) 仅用于管理端
}

// 默认大巴布局常量
const (
	defaultBusCols        = 5 // 左2列 + 过道1列 + 右2列
	defaultBusAisleCol    = 3 // 过道在第3列
	defaultSeatsPerRow    = 4 // 标准每排座位数（不含最后排）
	defaultLastRowSeats   = 5 // 最后排座位数
	maxValidateLayoutRows = 50 // 布局校验最大行数
	maxValidateLayoutCols = 20 // 布局校验最大列数
)

// SeatMapResponse 座位图API响应
type SeatMapResponse struct {
	Layout  SeatLayout   `json:"layout"`
	Seats   []SeatStatus `json:"seats"`   // 每个座位的状态
	Avail   int          `json:"avail"`   // 该区间可用座位数
}

// ValidateSeatLayout 校验座位布局JSON的合法性和合理性
// 返回 error 表示校验失败（含具体原因），nil 表示通过
func ValidateSeatLayout(layoutJSON string, seatCount int) error {
	if layoutJSON == "" {
		return nil // 空布局由 ParseSeatLayout 自动生成默认布局
	}
	var layout SeatLayout
	if err := json.Unmarshal([]byte(layoutJSON), &layout); err != nil {
		return fmt.Errorf("座位布局JSON格式错误")
	}
	if layout.Rows <= 0 || layout.Cols <= 0 {
		return fmt.Errorf("座位布局必须包含有效的行数和列数")
	}
	// 防止超大布局导致DoS：限制行数和列数上限
	if layout.Rows > maxValidateLayoutRows || layout.Cols > maxValidateLayoutCols {
		return fmt.Errorf("座位布局过大（最多%d行%d列）", maxValidateLayoutRows, maxValidateLayoutCols)
	}
	if len(layout.Cells) == 0 {
		return fmt.Errorf("座位布局必须包含格子定义")
	}
	// 统计座位类型格子数与 seatCount 对比
	actualSeats := 0
	for _, cell := range layout.Cells {
		if cell.Type == "seat" {
			actualSeats++
		}
	}
	if actualSeats != seatCount {
		return fmt.Errorf("座位布局中座位数(%d)与车辆座位数(%d)不一致", actualSeats, seatCount)
	}
	return nil
}

// ParseSeatLayout 解析座位布局JSON，空或非法则根据seatCount生成默认布局
func ParseSeatLayout(layoutJSON string, seatCount int) SeatLayout {
	if layoutJSON != "" {
		var layout SeatLayout
		if err := json.Unmarshal([]byte(layoutJSON), &layout); err == nil && len(layout.Cells) > 0 {
			return layout
		}
	}
	// 默认布局：标准大巴 2+2（左2 过道 右2），最后一排5座
	return GenerateDefaultLayout(seatCount)
}

// GenerateDefaultLayout 根据座位数自动生成标准大巴布局
// 左侧2列 + 过道1列 + 右侧2列 = 5列总宽
// 每排4座，最后排5座（有余数的话）
func GenerateDefaultLayout(seatCount int) SeatLayout {
	if seatCount <= 0 {
		return SeatLayout{}
	}

	cols := defaultBusCols       // 左2 + 过道 + 右2
	aisleCol := defaultBusAisleCol // 过道在第3列

	// 算排数：除了最后一排5座，前头每排4座
	var cells []SeatCell

	// 驾驶座在第一排第一列
	driverCell := SeatCell{Row: 1, Col: 1, Type: "driver"}
	cells = append(cells, driverCell)

	// 如果座位数<=默认最后排座位数，只有最后排
	if seatCount <= defaultLastRowSeats {
		lastRowSeats := seatCount
		for c := 1; c <= lastRowSeats; c++ {
			cells = append(cells, SeatCell{
				Row: 2, Col: c, Type: "seat", SeatNo: c,
			})
		}
		return SeatLayout{Rows: 2, Cols: cols, Cells: cells}
	}

	// 标准大巴：最后排N座，前面每排M座
	lastRowSeats := defaultLastRowSeats
	frontSeats := seatCount - lastRowSeats
	if frontSeats < 0 {
		lastRowSeats = seatCount
		frontSeats = 0
	}
	// 确保最后排至少1座
	if lastRowSeats < 1 {
		lastRowSeats = seatCount
		frontSeats = 0
	}

	frontRows := int(math.Ceil(float64(frontSeats) / float64(defaultSeatsPerRow)))
	totalRows := frontRows + 1 // +1排最后排
	// 驾驶座占第1排，实际座位从第1排开始
	seatNo := 0

	for r := 1; r <= frontRows; r++ {
		for c := 1; c <= cols; c++ {
			// 跳过驾驶座位置（第1排第1列）
			if r == 1 && c == 1 {
				continue
			}
			if c == aisleCol {
				// 过道列
				cells = append(cells, SeatCell{Row: r, Col: c, Type: "aisle"})
			} else if c >= 1 && c <= cols && c != aisleCol && seatNo < frontSeats {
				seatNo++
				cells = append(cells, SeatCell{
					Row: r, Col: c, Type: "seat", SeatNo: seatNo,
				})
			} else {
				cells = append(cells, SeatCell{Row: r, Col: c, Type: "empty"})
			}
		}
	}

	// 最后排：c=1到lastRowSeats都是座位（可能不满5列）
	lastRow := totalRows
	for c := 1; c <= cols; c++ {
		if c <= lastRowSeats {
			seatNo++
			cells = append(cells, SeatCell{
				Row: lastRow, Col: c, Type: "seat", SeatNo: seatNo,
			})
		} else {
			cells = append(cells, SeatCell{Row: lastRow, Col: c, Type: "empty"})
		}
	}

	return SeatLayout{Rows: totalRows, Cols: cols, Cells: cells}
}

// GetSeatMap 获取班次某区间的座位图（含占用状态）
// 必须在事务内或至少读已提交隔离级别下调用
func GetSeatMap(tx *gorm.DB, tripID uint, totalSeats int, fromOrder, toOrder int, layout SeatLayout) (*SeatMapResponse, error) {
	// 这趟车这段路座位都被谁占了
	assignments, err := querySeatAssignments(tx, tripID, true)
	if err != nil {
		return nil, err
	}

	// 座位占用映射：seatNo -> 这段路被占木有
	occupiedMap := make(map[int]bool)
	occupiedNameMap := make(map[int]string)
	for _, a := range assignments {
		seatNum := 0
		for _, ch := range a.SeatNo {
			if ch >= '0' && ch <= '9' {
				seatNum = seatNum*10 + int(ch-'0')
			}
		}
		if seatNum == 0 {
			continue
		}
		// 区间重叠检查
		if fromOrder < a.ToOrder && toOrder > a.FromOrder {
			occupiedMap[seatNum] = true
			// 脱敏姓名
			name := a.Name
			if len([]rune(name)) > 1 {
				runes := []rune(name)
				name = string(runes[0]) + "*"
			}
			occupiedNameMap[seatNum] = name
		}
	}

	// 拼响应
	var seats []SeatStatus
	occupiedCount := 0
	for _, cell := range layout.Cells {
		if cell.Type != "seat" {
			continue
		}
		isOccupied := occupiedMap[cell.SeatNo]
		status := SeatStatus{
			SeatNo:   cell.SeatNo,
			Row:      cell.Row,
			Col:      cell.Col,
			Occupied: isOccupied,
		}
		if isOccupied {
			occupiedCount++
			status.OccupiedBy = occupiedNameMap[cell.SeatNo]
		}
		seats = append(seats, status)
	}

	avail := totalSeats - occupiedCount
	if avail < 0 {
		avail = 0
	}

	return &SeatMapResponse{
		Layout: layout,
		Seats:  seats,
		Avail:  avail,
	}, nil
}

// AssignSeatsWithPreferences 分配座位（优先使用用户选择的座位号）
// seatPrefs: 用户期望的座位号列表，可为空（向后兼容自动分配）
// 返回值: assignedSeats=分配结果, allHonored=全部偏好被满足（无偏好时也为true）, error
func AssignSeatsWithPreferences(tx *gorm.DB, tripID uint, totalSeats, fromOrder, toOrder, passengerCount int, seatPrefs []int) ([]string, bool, error) {
	// 如果用户有选座偏好，逐个验证并锁定
	if len(seatPrefs) > 0 {
		// 去重 + 边界校验：座位号必须在 [1, totalSeats] 范围内，且不能重复（同一座位不能分给两人）
		seen := make(map[int]bool, len(seatPrefs))
		for _, pref := range seatPrefs {
			if pref < 1 || pref > totalSeats || seen[pref] {
				// 无效座位号或重复座位号，回退到自动分配
				seats, err := AssignSeats(tx, tripID, totalSeats, fromOrder, toOrder, passengerCount)
				return seats, false, err
			}
			seen[pref] = true
		}
		// 数量校验：偏好座位数必须与乘客数一致，否则按自动分配处理（防止多分/少分座位）
		if len(seatPrefs) != passengerCount {
			seats, err := AssignSeats(tx, tripID, totalSeats, fromOrder, toOrder, passengerCount)
			return seats, false, err
		}
		// 先获取已分配座位
		assignments, err := querySeatAssignments(tx, tripID, false)
		if err != nil {
			return nil, false, err
		}

		// 看看选的座还中不中
		var result []string
		allAvailable := true
		for _, pref := range seatPrefs {
			prefStr := strconv.Itoa(pref)
			available := true
			for _, a := range assignments {
				aSeatNo := 0
				for _, ch := range a.SeatNo {
					if ch >= '0' && ch <= '9' {
						aSeatNo = aSeatNo*10 + int(ch-'0')
					}
				}
				if aSeatNo == pref && fromOrder < a.ToOrder && toOrder > a.FromOrder {
					available = false
					break
				}
			}
			if !available {
				allAvailable = false
				break
			}
			result = append(result, prefStr)
		}
		if allAvailable && len(result) >= passengerCount {
			return result, true, nil
		}
		// 偏好座位不足或被占用，回退到自动分配，标记 allHonored=false
		seats, err := AssignSeats(tx, tripID, totalSeats, fromOrder, toOrder, passengerCount)
		return seats, false, err
	}

	// 无偏好，自动分配
	seats, err := AssignSeats(tx, tripID, totalSeats, fromOrder, toOrder, passengerCount)
	return seats, true, err
}
