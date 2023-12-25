package dal

import "testing"

func TestInitDB(t *testing.T) {

	myCfg := DBConfig{
		Username:     "alice",
		Password:     "123456",
		Address:      "127.0.0.1:3306",
		DatabaseName: "abe_mining_pool",
	}

	t.Run("test_1", func(t *testing.T) {
		if err := InitDB(&myCfg, true); err != nil {
			t.Errorf("InitDB() error = %v", err)
		}
	})
}
