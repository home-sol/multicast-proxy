package reflector

type MacAddress string

type Config struct {
	NetInterface     string `mapstructure:"net_interface"`
	WindowsInterface string `mapstructure:"windows_net_interface"`
	Devices          map[MacAddress]Device
}

type Device struct {
	OriginPool  uint16   `mapstructure:"origin_pool"`
	SharedPools []uint16 `mapstructure:"shared_pools"`
}
