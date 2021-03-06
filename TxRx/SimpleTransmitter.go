package TxRx

import (
	"fmt"

	// "github.com/wiless/cellular"
	// "github.com/wiless/cellular/channel"
	"github.com/wiless/gocomm"
	// "github.com/wiless/gocomm/chipset"
	"github.com/wiless/gocomm/core"
	// "github.com/wiless/gocomm/modem"
	// "github.com/wiless/gocomm/sink"
	"github.com/wiless/vlib"

	"log"
	"math/rand"
	"sync"
	// "os"

	"time"
)

var matlab *vlib.Matlab

func init() {
	matlab = vlib.NewMatlab("channel")
	matlab.Silent = true
	matlab.Json = true
	rand.Seed(time.Now().Unix())

}

func main() {
	starttime := time.Now()

	Ntx := 4
	tx := make([]SimpleTransmitter, Ntx)
	wg := new(sync.WaitGroup)

	for i := 0; i < Ntx; i++ {
		wg.Add(1)
		tx[i].Init()
		tx[i].SetID(i)
		tx[i].Nblocks = 1
		tx[i].SetWaitGroup(wg)
		go tx[i].StartTransmit()

	}

	for i := 0; i < Ntx; i++ {

		go func(txch gocomm.Complex128AChannel) {
			for cnt := 0; ; cnt++ {
				chdata := <-txch
				// fmt.Printf("\n ,Tx %s : %f", chdata.Message, chdata.Ch)
				if chdata.GetMaxExpected()-1 == cnt {
					break
				}
			}
		}(tx[i].GetChannel())

	}
	wg.Wait()
	matlab.Close()
	fmt.Printf("I am done %s \n", time.Since(starttime))
}

type SimpleTransmitter struct {
	nid      int
	sch      gocomm.Complex128AChannel
	proxyPin gocomm.Complex128AChannel
	Nblocks  int
	BlockLen int
	wg       *sync.WaitGroup
	// All Core Chips
	Txmodem *core.Modem
	seed    uint32
	key     string

	probes []gocomm.Complex128AChannel
}

func (s *SimpleTransmitter) SetID(id int) {
	s.nid = id
}

func (s *SimpleTransmitter) NProbes() int {
	return len(s.probes)
}

func (s SimpleTransmitter) GetChannel() gocomm.Complex128AChannel {
	if s.proxyPin == nil {
		return s.sch
	}
	return s.proxyPin
}

///
func (s *SimpleTransmitter) SetProxyPin(proxypin gocomm.Complex128AChannel) {
	s.proxyPin = proxypin
}

func (s *SimpleTransmitter) Init() {
	s.sch = gocomm.NewComplex128AChannel()
	s.proxyPin = nil
	s.Nblocks = 10
	s.BlockLen = 32 // No of bits
	s.seed = rand.Uint32()
	s.key = string(vlib.RandString(5))

	s.Txmodem = new(core.Modem)
	s.Txmodem.SetName(s.Key() + "_MODEM")

	s.Txmodem.Init(2, "QPSK")
	s.Txmodem.InitializeChip()

	s.probes = make([]gocomm.Complex128AChannel, 1)
	s.probes[0] = gocomm.NewComplex128AChannel()

}

func (s *SimpleTransmitter) Key() string {
	return s.key
}

func (s *SimpleTransmitter) GetProbe(prbId int) gocomm.Complex128AChannel {
	if prbId >= s.NProbes() {
		log.Panicln("Tx:GetProbe Index out of bound")
	}
	return s.probes[prbId]
}

func (s *SimpleTransmitter) CreateCircuit() {

}

func (s *SimpleTransmitter) StartTransmit() {

	if s.sch == nil || s.wg == nil {
		log.Panicln("SimpleTransmitter Not Initialized or No WaitGroup set !! No channel yet")
	}

	// log.Println("Ready to send ??")
	var chdata gocomm.SComplex128AObj
	chdata.MaxExpected = s.Nblocks
	chdata.Message = s.key
	chdata.Ts = 1
	N := s.BlockLen // 32bits=16SYMBOLS per TTI

	// log.Println("Transmitter: Ready to send ??")
	chdata.TimeStamp = -1
	for i := 0; i < s.Nblocks; i++ {

		/// Modulation data

		chdata.Next(s.Txmodem.ModulateBits(vlib.RandB(N)))
		log.Printf("Transmitter %d , @ TimeStamp : %f : Writing (%d)symbols into Go-chan ", s.GetID(), chdata.TimeStamp, len(chdata.Ch))
		// Do other transmitter processing like CDMA/OFDM etc.if applicable

		// Finally write to output Pin of Transmitter
		s.sch <- chdata
		///write to probe
		select {
		case s.probes[0] <- chdata:
			log.Println("===========Tx wrote to Probe===========")
		default:
			log.Println("Unable to write to probe")
		}

	}

	if s.wg != nil {
		log.Println("Done Transmission job of all Nblocks of data ", s.GetID())
		s.wg.Done()
	}

}

func (s SimpleTransmitter) GetID() int {
	return s.nid
}

func (s SimpleTransmitter) GetSeed() int64 {
	return int64(s.seed)
}
func (s SimpleTransmitter) IsActive() bool {
	return true
}
func (s *SimpleTransmitter) SetWaitGroup(wg *sync.WaitGroup) {
	s.wg = wg
}
