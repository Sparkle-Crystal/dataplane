package worker

import (
	permissions "dataplane/mainapp/auth_permissions"
	"dataplane/mainapp/database/models"
	"dataplane/mainapp/logging"
	"dataplane/mainapp/messageq"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/websocket/v2"
	"github.com/nats-io/nats.go"
)

// var Broadcast1 = make(chan []byte)

type MsgResult1 struct {
	Message []byte
	Err     error
}

var messagereceive1 = make(chan MsgResult)
var disconnectConn1 = make(chan string)

// https://github.com/gorilla/websocket/blob/master/examples/chat/client.go

// https://github.com/marcelo-tm/testws/blob/master/main.go
func RoomUpdates(conn *websocket.Conn, environmentID string, subject string, id string) {

	// ---- Permissions and security checks

	currentUser := conn.Locals("currentUser").(string)
	platformID := conn.Locals("platformID").(string)
	room := ""

	// ----- Permissions
	perms := []models.Permissions{
		{Resource: "admin_platform", ResourceID: platformID, Access: "write", Subject: "user", SubjectID: currentUser, EnvironmentID: "d_platform"},
		{Resource: "admin_environment", ResourceID: environmentID, Access: "write", Subject: "user", SubjectID: currentUser, EnvironmentID: environmentID},
	}

	switch subject {
	case "taskupdate." + environmentID + "." + id:
		fmt.Println("one")
		room = "pipeline-run-updates"
	default:
		log.Println("subject not found")
		return
	}

	permOutcome, _, _, _ := permissions.MultiplePermissionChecks(perms)

	if permOutcome == "denied" {
		logging.PrintSecretsRedact("Requires permissions")
		return
	}

	sub, _ := messageq.NATSencoded.Subscribe(subject, func(m *nats.Msg) {

		broadcastq <- message{room: room, data: m.Data}

	})

	// When the function returns, unregister the client and close the connection
	defer func() {
		unregisterq <- subscription{conn: conn, room: room}
		conn.Close()
		sub.Unsubscribe()
	}()

	// Register the client
	registerq <- subscription{conn: conn, room: room}

	// go SecureTimeout()

	for {

		mt, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("read error:", err)
			}
			return
		}

		if os.Getenv("messagedebug") == "true" {
			logging.PrintSecretsRedact("message received from client:", mt, string(message))
		}

	}

}