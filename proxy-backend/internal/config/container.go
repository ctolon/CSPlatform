package config

// containerEditConfig holds the configuration for the container. CONTAINER_ALLOW_EDIT_NAME
type containerEditConfig struct {
	ContainerAllowEditName       bool `mapstructure:"CONTAINER_ALLOW_EDIT_NAME"`
	ContainerAllowEditCPU        bool `mapstructure:"CONTAINER_ALLOW_EDIT_CPU"`
	ContainerAllowEditEnv        bool `mapstructure:"CONTAINER_ALLOW_EDIT_ENV"`
	ContainerAllowEditExpose     bool `mapstructure:"CONTAINER_ALLOW_EDIT_EXPOSE"`
	ContainerAllowEditExtraHosts bool `mapstructure:"CONTAINER_ALLOW_EDIT_EXTRA_HOSTS"`
	ContainerAllowEditImage      bool `mapstructure:"CONTAINER_ALLOW_EDIT_IMAGE"`
	ContainerAllowEditMemory     bool `mapstructure:"CONTAINER_ALLOW_EDIT_MEMORY"`
	ContainerAllowEditNetwork    bool `mapstructure:"CONTAINER_ALLOW_EDIT_NETWORK"`
	ContainerAllowEditPorts      bool `mapstructure:"CONTAINER_ALLOW_EDIT_PORTS"`
	ContainerAllowEditRestart    bool `mapstructure:"CONTAINER_ALLOW_EDIT_RESTART"`
	ContainerAllowEditSysctls    bool `mapstructure:"CONTAINER_ALLOW_EDIT_SYSCTLS"`
	ContainerAllowEditVolumes    bool `mapstructure:"CONTAINER_ALLOW_EDIT_VOLUMES"`
}
