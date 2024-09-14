package models

type Data struct {
	Users   []*User
	Offices []*Office
}

func NewData() *Data {
	user := NewUser("Rowan")
	return &Data{
		Users: []*User{
			user,
		},
		Offices: []*Office{
			{
				Name: "Cambridge",
				Code: "ABCDEF",
				Players: []*Player{
					newPlayer(user.Username),
				},
				Admin: user,
			},
		},
	}
}

func (d *Data) GetUser(username string) *User {
	for _, u := range d.Users {
		if u.Username == username {
			return u
		}
	}

	return nil
}

func (d *Data) CreateUser(username string) {
	d.Users = append(d.Users, NewUser(username))
}

func (d *Data) CreateOffice(name string, user *User) {
	d.Offices = append(d.Offices, NewOffice(name, user))
}

func (d *Data) FindOfficeByName(name string) *Office {
	for _, o := range d.Offices {
		if o.Name == name {
			return o
		}
	}

	return nil
}

func (d *Data) FindOfficeByCode(code string) *Office {
	for _, o := range d.Offices {
		if o.Code == code {
			return o
		}
	}

	return nil
}

func (d *Data) GetUserOffices(user *User) []*Office {
	offices := []*Office{}
	for _, o := range d.Offices {
		for _, player := range o.Players {
			if player.Name == user.Username {
				offices = append(offices, o)
				break
			}
		}
	}

	return offices
}
