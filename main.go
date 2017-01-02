package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/Philipp15b/go-steam/steamid"
	"github.com/golang/protobuf/proto"
	"github.com/samcday/csgo-demo-parser/rank"
	camelcase "github.com/segmentio/go-camelcase"
	"github.com/stegmannc/csgo-demoparser"
	"github.com/stegmannc/csgo-demoparser/protom"
)

const valveMagicNumber = 76561197960265728

func readInt(r *bufio.Reader) uint32 {
	var i uint32
	err := binary.Read(r, binary.LittleEndian, &i)
	if err != nil {
		panic(err)
	}
	return i
}

func readFloat(r *bufio.Reader) float32 {
	var f float32
	err := binary.Read(r, binary.LittleEndian, &f)
	if err != nil {
		panic(err)
	}
	return f
}

func readString(r *bufio.Reader) string {
	s, err := r.ReadString(0)
	if err != nil {
		panic(err)
	}
	return s
}

func mapGameEventKeyValue(valueType int32, key *protom.CSVCMsg_GameEventKeyT) interface{} {
	switch valueType {
	case 1:
		return key.GetValString()
	case 2:
		return key.GetValFloat()
	case 3:
		return key.GetValLong()
	case 4:
		return key.GetValShort()
	case 5:
		return key.GetValByte()
	case 6:
		return key.GetValBool()
	default:
		return nil
	}
}

func main() {
	f, err := os.Open("match730_003180245891948740689_1625585345_115.dem")
	if err != nil {
		panic(err)
	}

	stream := demoinfo.NewDemoStream(f, -1)
	header := &demoinfo.DemoHeader{}

	if err = binary.Read(stream, binary.LittleEndian, header); err != nil {
		panic(err)
	}

	if string(header.Demofilestamp[:7]) != "HL2DEMO" {
		panic(fmt.Errorf("Unexpected file header %s", string(header.Demofilestamp[:7])))
	}

	context := demoinfo.NewDemoContext(header)
	players := make(map[steamid.SteamId]string)

	for {
		cmdHeader := demoinfo.DemoCmdHeader{
			Cmd:        stream.GetUInt8(),
			Tick:       stream.GetInt(),
			Playerslot: stream.GetUInt8(),
		}
		switch cmdHeader.Cmd {
		case demoinfo.DemSignon, demoinfo.DemPacket:
			stream.Skip(demoinfo.PacketOffset)

			packetStream := stream.CreatePacketStream()
			for !packetStream.IsProcessed() {
				messageType := packetStream.GetVarInt()
				length := packetStream.GetVarInt()

				switch protom.SVC_Messages(messageType) {
				case protom.SVC_Messages_svc_GameEventList:
					msg := &protom.CSVCMsg_GameEventList{}
					packetStream.ParseToStruct(msg, length)
					context.GameEventList = msg
				case protom.SVC_Messages_svc_GameEvent:
					msg := &protom.CSVCMsg_GameEvent{}
					packetStream.ParseToStruct(msg, length)
					descriptor := context.GetGameEventDescriptor(msg.GetEventid())
					event := demoinfo.NewDemoGameEvent(msg.GetEventid(), descriptor.GetName(), cmdHeader.Tick)
					descriptorKeys := descriptor.GetKeys()
					eventKeys := msg.GetKeys()

					for i, eventKey := range eventKeys {
						descriptorKey := descriptorKeys[i]
						name := camelcase.Camelcase(descriptorKey.GetName())
						mappedValue := mapGameEventKeyValue(descriptorKey.GetType(), eventKey)
						event.Data[name] = mappedValue
					}

					if event.Name == "player_connect" {
						rawID := event.Data["networkid"].(string)
						if rawID != "BOT" {
							steamID, err := steamid.NewId(rawID)
							if err != nil {
								panic(err)
							}
							if _, ok := players[steamID]; !ok {
								players[steamID] = event.Data["name"].(string)
							}
						}
					}

				case protom.SVC_Messages_svc_UserMessage:
					msg := new(protom.CSVCMsg_UserMessage)
					packetStream.ParseToStruct(msg, length)
					if msg.GetMsgType() == 52 {
						r := new(rank.CCSUsrMsg_ServerRankUpdate)
						proto.Unmarshal(msg.GetMsgData(), r)

						for _, item := range r.RankUpdate {
							id := steamid.SteamId(uint64(*item.AccountId) + valveMagicNumber)
							fmt.Printf("%s is rank %v\n", players[id], item.GetRankNew())
						}
					}
				default:
					packetStream.Skip(int64(length))
				}

			}
		case demoinfo.DemDatatables:
			stream.Skip(int64(stream.GetInt()))
		case demoinfo.DemSringTables:
			stream.Skip(int64(stream.GetInt()))
		case demoinfo.DemStop:
			fmt.Println("STOP")
			return
		case demoinfo.DemSynctick:
			continue
		default:
			panic(fmt.Errorf("Uhhh"))
		}
	}
}
