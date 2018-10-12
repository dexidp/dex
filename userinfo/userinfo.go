package userinfo


type Userinfo interface{
	Close()
	
	Authenticate() error
	GetUserInformation() error
}
