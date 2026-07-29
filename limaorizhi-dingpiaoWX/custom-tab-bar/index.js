Component({
  data: {
    selected: 0,
    color: "#000",
    selectedColor: "#000",
    list: [
      {
        pagePath: "/pages/home/home",
        text: "首页",
        iconPath: "/images/tabbar/home.svg",
        selectedIconPath: "/images/tabbar/home-active.svg"
      },
      {
        pagePath: "/pages/search/search",
        text: "订单",
        iconPath: "/images/tabbar/order.svg",
        selectedIconPath: "/images/tabbar/order-active.svg"
      },
      {
        pagePath: "/pages/mine/mine",
        text: "我的",
        iconPath: "/images/tabbar/mine.svg",
        selectedIconPath: "/images/tabbar/mine-active.svg"
      }
    ]
  },
  methods: {
    switchTab(e) {
      const data = e.currentTarget.dataset
      const url = data.path
      wx.switchTab({
        url,
        success: () => {
          this.setData({
            selected: data.index
          })
        }
      })
    }
  }
})
