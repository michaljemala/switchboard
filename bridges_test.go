package switchboard_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/switchboard"
	"github.com/pivotal-cf-experimental/switchboard/fakes"
	"github.com/pivotal-golang/lager"
)

var _ = Describe("Bridges", func() {
	var bridges Bridges
	var bridge1 *fakes.FakeBridge
	var bridge2 *fakes.FakeBridge
	var bridge3 *fakes.FakeBridge

	BeforeEach(func() {
		bridges = NewBridges()

		bridge1 = &fakes.FakeBridge{}
		bridge2 = &fakes.FakeBridge{}
		bridge3 = &fakes.FakeBridge{}

		bridges.Add(bridge1)
		bridges.Add(bridge2)
		bridges.Add(bridge3)
	})

	Describe("Concurrent operations", func() {
		It("do not result in a race", func() {
			readySetGo := make(chan interface{})

			doneChans := []chan interface{}{
				make(chan interface{}),
				make(chan interface{}),
				make(chan interface{}),
				make(chan interface{}),
				make(chan interface{}),
			}

			go func() {
				<-readySetGo
				bridges.Add(&fakes.FakeBridge{})
				close(doneChans[0])
			}()

			go func() {
				<-readySetGo
				bridges.Contains(bridge1)
				close(doneChans[1])
			}()

			go func() {
				<-readySetGo
				bridges.Remove(bridge2)
				close(doneChans[2])
			}()

			go func() {
				<-readySetGo
				bridges.Size()
				close(doneChans[3])
			}()

			go func() {
				<-readySetGo
				bridges.RemoveAndCloseAll()
				close(doneChans[4])
			}()

			close(readySetGo)

			for _, done := range doneChans {
				<-done
			}
		})
	})

	Describe("Remove", func() {
		It("removes only the given bridge", func() {
			err := bridges.Remove(bridge2)
			Expect(err).NotTo(HaveOccurred())

			Expect(bridges.Contains(bridge1)).To(BeTrue())
			Expect(bridges.Contains(bridge2)).To(BeFalse())
			Expect(bridges.Contains(bridge3)).To(BeTrue())

			Expect(bridges.Size()).To(Equal(2))
		})

		Context("when the bridge cannot be found", func() {
			It("returns an error", func() {
				err := bridges.Remove(NewConnectionBridge(&fakes.FakeReadWriteCloser{}, &fakes.FakeReadWriteCloser{}, lager.NewLogger("test")))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("Bridge not found"))
			})
		})
	})

	Describe("RemoveAndCloseAll", func() {
		It("closes all bridges", func() {
			bridges.RemoveAndCloseAll()

			Expect(bridge1.CloseCallCount()).To(Equal(1))
			Expect(bridge2.CloseCallCount()).To(Equal(1))
			Expect(bridge3.CloseCallCount()).To(Equal(1))
		})

		It("removes all bridges", func() {
			bridges.RemoveAndCloseAll()

			Expect(bridges.Size()).To(Equal(0))
		})
	})
})
