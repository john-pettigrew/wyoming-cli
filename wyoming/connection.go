package wyoming

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

var DescribeMessageType string = "describe"

type WyomingMessage struct {
	Type          string      `json:"type"`
	Version       string      `json:"version"`
	Data          interface{} `json:"data"`
	DataLength    int         `json:"data_length,omitempty"`
	PayloadLength int         `json:"payload_length,omitempty"`
}

type WyomingMessageContainer struct {
	Message WyomingMessage
	Data    []byte
	Payload []byte
}

type WyomingConnection struct {
	Conn          net.Conn
	ServerAddr    string
	VoiceServices WyomingVoiceServicesData
}

type WyomingAttribution struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type WyomingVoiceServicesData struct {
	TTS []WyomingVoiceServicesTTSData `json:"tts,omitempty"`
	ASR []WyomingVoiceServicesASRData `json:"asr,omitempty"`
}

// Disconnect disconnects from the Wyoming server.
func (w *WyomingConnection) Disconnect() error {
	return w.Conn.Close()
}

// GetAvailableServices returns the voice services reported to be supported by the
// Wyoming server.
func (w *WyomingConnection) GetAvailableServices() (WyomingVoiceServicesData, error) {
	err := w.SendMessage(WyomingMessage{Type: DescribeMessageType})
	if err != nil {
		return WyomingVoiceServicesData{}, err
	}

	reader := bufio.NewReaderSize(w.Conn, 1024*1024)
	newMsg, err := w.ReceiveMessageUsingReader(reader)
	if err != nil {
		return WyomingVoiceServicesData{}, err
	}

	var voiceServices WyomingVoiceServicesData
	err = json.Unmarshal(newMsg.Data, &voiceServices)
	if err != nil {
		return WyomingVoiceServicesData{}, err
	}

	return voiceServices, nil
}

// SendMessage sends a message to a Wyoming server followed by a newline character.
func (w *WyomingConnection) SendMessage(msg WyomingMessage) error {
	jsonMessage, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w.Conn, "%s\n", string(jsonMessage))
	if err != nil {
		return err
	}

	return nil
}

// SendMessageContainer sends Message, Data, and Payload from container to a Wyoming server. DataLength and PayloadLength
// are set before sending.
func (w *WyomingConnection) SendMessageContainer(container WyomingMessageContainer) error {
	if container.Data != nil {
		container.Message.DataLength = len(container.Data)
	}
	if container.Payload != nil {
		container.Message.PayloadLength = len(container.Payload)
	}

	err := w.SendMessage(container.Message)
	if err != nil {
		return err
	}

	if container.Message.DataLength > 0 {
		_, err = w.Conn.Write(container.Data)
		if err != nil {
			return err
		}
	}

	if container.Message.PayloadLength > 0 {
		_, err = w.Conn.Write(container.Payload)
		if err != nil {
			return err
		}
	}

	return nil
}

// ReceiveMessage receives a message from a Wyoming server.
func (w *WyomingConnection) ReceiveMessage() (WyomingMessageContainer, error) {
	reader := bufio.NewReader(w.Conn)
	return w.ReceiveMessageUsingReader(reader)
}

// ReceiveMessageUsingReader receives a message from a Wyoming server using reader. This can
// be helpful if multiple messages are being read or if a larger buffer size is needed.
func (w *WyomingConnection) ReceiveMessageUsingReader(reader *bufio.Reader) (WyomingMessageContainer, error) {
	res := WyomingMessageContainer{Message: WyomingMessage{}}

	msgStr, err := reader.ReadString('\n')
	if err != nil {
		return WyomingMessageContainer{}, err
	}

	err = json.Unmarshal([]byte(msgStr), &res.Message)
	if err != nil {
		return WyomingMessageContainer{}, err
	}

	if res.Message.DataLength > 0 {
		messageData, err := reader.Peek(res.Message.DataLength)
		if err != nil {
			return WyomingMessageContainer{}, err
		}

		res.Data = make([]byte, len(messageData))
		copy(res.Data, messageData)

		_, err = reader.Discard(res.Message.DataLength)
		if err != nil {
			return WyomingMessageContainer{}, err
		}
	}

	if res.Message.PayloadLength > 0 {
		payloadData, err := reader.Peek(res.Message.PayloadLength)
		if err != nil {
			return WyomingMessageContainer{}, err
		}

		res.Payload = make([]byte, len(payloadData))
		copy(res.Payload, payloadData)

		_, err = reader.Discard(res.Message.PayloadLength)
		if err != nil {
			return WyomingMessageContainer{}, err
		}
	}

	return res, nil
}

// Reconnect disconnects then reconnects to a Wyoming server using ServerAddr
func (w *WyomingConnection) Reconnect() error {
	err := w.Disconnect()
	if err != nil {
		return err
	}
	newConn, err := net.Dial("tcp", w.ServerAddr)
	if err != nil {
		return err
	}

	w.Conn = newConn
	return nil
}

// Connect connects to a Wyoming server and checks supported features.
func Connect(serverAddr string) (WyomingConnection, error) {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return WyomingConnection{}, err
	}

	w := WyomingConnection{ServerAddr: serverAddr, Conn: conn}
	w.VoiceServices, err = w.GetAvailableServices()
	if err != nil {
		return WyomingConnection{}, err
	}

	return w, nil
}
