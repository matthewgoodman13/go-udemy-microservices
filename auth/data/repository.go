package data

type Repository interface {
	// GetAll returns a slice of all users, sorted by last name
	GetAll() ([]*User, error)

	// GetByEmail returns one user by email
	GetByEmail(email string) (*User, error)

	// GetOne returns one user by id
	GetOne(id int) (*User, error)

	// Update updates a user in the database
	Update(user User) error

	// DeleteById deletes a user from the database
	DeleteByID(id int) error

	// Insert inserts a new user into the database
	Insert(user User) (int, error)

	// ResetPassword
	ResetPassword(password string, user User) error

	// PasswordMatches
	PasswordMatches(plainText string, user User) (bool, error)
}
