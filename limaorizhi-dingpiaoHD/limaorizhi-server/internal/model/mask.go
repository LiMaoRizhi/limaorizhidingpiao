package model

import "limaorizhi-server/internal/pkg/idcard"

// 脱敏方法：查询后、响应前显式调用
// 不在 AfterFind 中脱敏，避免 BeforeUpdate 把脱敏值写回数据库

// 乘客 / 常用乘客

func (p *OrderPassenger) Mask() {
	if p == nil {
		return
	}
	p.IDCardNo = idcard.MaskIDCard(p.IDCardNo)
	p.Phone = idcard.MaskPhone(p.Phone)
}

func MaskPassengers(list []OrderPassenger) {
	for i := range list {
		list[i].Mask()
	}
}

// MaskPassengersKeepPhone 仅脱敏身份证号，保留手机号（司机端用）
func MaskPassengersKeepPhone(list []OrderPassenger) {
	for i := range list {
		if list[i].IDCardNo != "" {
			list[i].IDCardNo = idcard.MaskIDCard(list[i].IDCardNo)
		}
		// 不脱敏 Phone，司机需要联系电话乘客
	}
}

func (p *Passenger) Mask() {
	if p == nil {
		return
	}
	p.IDCardNo = idcard.MaskIDCard(p.IDCardNo)
	p.Phone = idcard.MaskPhone(p.Phone)
}

func MaskPassengerList(list []Passenger) {
	for i := range list {
		list[i].Mask()
	}
}

// 订单

func (o *Order) Mask() {
	if o == nil {
		return
	}
	o.ContactPhone = idcard.MaskPhone(o.ContactPhone)
	o.SenderPhone = idcard.MaskPhone(o.SenderPhone)
	o.ReceiverPhone = idcard.MaskPhone(o.ReceiverPhone)
}

func MaskOrders(list []Order) {
	for i := range list {
		list[i].Mask()
	}
}

// 用户

func (u *User) Mask() {
	if u == nil {
		return
	}
	u.Phone = idcard.MaskPhone(u.Phone)
	u.OpenID = ""
	u.UnionID = ""
}

func MaskUsers(list []User) {
	for i := range list {
		list[i].Mask()
	}
}
