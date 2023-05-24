package session

import "testing"

var (
	user1 = &User{"Tom", 18}
	user2 = &User{"Sam", 25}
	user3 = &User{"Jack", 25}
)

func testRecordInit(t *testing.T) *Session {
	t.Helper()

	session := NewSession().Model(&User{})
	err1 := session.DropTable()
	err2 := session.CreateTable()
	_, err3 := session.Insert(user1, user2)

	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatal("failed init test records")
	}

	return session
}

func TestInsert(t *testing.T) {
	session := testRecordInit(t)

	affected, err := session.Insert(user3)
	if err != nil || affected != 1 {
		t.Fatal("failed to create record")
	}
}

func TestFind(t *testing.T) {
	session := testRecordInit(t)

	var users []User
	err := session.Find(&users)
	if err != nil || len(users) != 2 {
		t.Fatal("failed to query all")
	}
}