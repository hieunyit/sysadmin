package application

type Services struct {
	User               *UserService
	Group              *GroupService
	OpenVPNAdmin       *OpenVPNAdminService
	OpenVPNProvisioner *OpenVPNProvisioningService
}
