package commands_test

import (
	"errors"
	"provisioner/provisioner"
	"provisioner/provisioner/commands"
	"provisioner/provisioner/mocks"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
	"provisioner/fs"
)

var _ = Describe("ConfigureDnsmasq", func() {
	var (
		mockCtrl      *gomock.Controller
		mockFS        *mocks.MockFS
		mockCmdRunner *mocks.MockCmdRunner
		cDnsmasq      *commands.ConfigureDnsmasq
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockFS = mocks.NewMockFS(mockCtrl)
		mockCmdRunner = mocks.NewMockCmdRunner(mockCtrl)
		cDnsmasq = &commands.ConfigureDnsmasq{
			FS:         mockFS,
			CmdRunner:  mockCmdRunner,
			Domain:     "some-domain",
			ExternalIP: "some-external-ip",
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("#Run", func() {
		Context("when there are external nameservers in /etc/resolv.conf", func() {
			It("should write dns resolutions for consul and domain into the dnsmasq config and save external nameservers", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("resolvconf", "--disable-updates"),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "stop"),
					mockCmdRunner.EXPECT().Output("ip", "route", "get", "1").Return([]byte("some-ip via some-other-ip dev eth0  src some-internal-ip\n cache"), nil),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/domain", strings.NewReader("address=/.some-domain/some-external-ip\naddress=/.cf.internal/127.0.0.1"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/interface", strings.NewReader("listen-address=some-internal-ip"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.conf", strings.NewReader("resolv-file=/var/pcfdev/external-resolv.conf"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Exists("/var/pcfdev/external-resolv.conf").Return(false, nil),
					mockFS.EXPECT().Read("/etc/resolv.conf").Return([]byte("nameserver 127.0.0.1\nnameserver some-external-nameserver\n# Generated by bosh-agent\nnameserver some-other-external-nameserver"), nil),
					mockFS.EXPECT().Write("/var/pcfdev/external-resolv.conf", strings.NewReader("nameserver some-external-nameserver\nnameserver some-other-external-nameserver"), os.FileMode(fs.FileModeRootReadWrite)),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "start"),
					mockFS.EXPECT().Write("/etc/resolv.conf", strings.NewReader("nameserver some-internal-ip"), os.FileMode(fs.FileModeRootReadWrite)),
				)

				Expect(cDnsmasq.Run()).To(Succeed())
			})
		})

		Context("when there are no external nameservers in /etc/resolv.conf", func() {
			It("should write dns resolutions for consul and domain into the dnsmasq config and save external nameservers", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("resolvconf", "--disable-updates"),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "stop"),
					mockCmdRunner.EXPECT().Output("ip", "route", "get", "1").Return([]byte("some-ip via some-other-ip dev eth0  src some-internal-ip\n cache"), nil),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/domain", strings.NewReader("address=/.some-domain/some-external-ip\naddress=/.cf.internal/127.0.0.1"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/interface", strings.NewReader("listen-address=some-internal-ip"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.conf", strings.NewReader("resolv-file=/var/pcfdev/external-resolv.conf"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Exists("/var/pcfdev/external-resolv.conf").Return(false, nil),
					mockFS.EXPECT().Read("/etc/resolv.conf").Return([]byte("nameserver 127.0.0.1"), nil),
					mockFS.EXPECT().Write("/var/pcfdev/external-resolv.conf", strings.NewReader(""), os.FileMode(fs.FileModeRootReadWrite)),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "start"),
					mockFS.EXPECT().Write("/etc/resolv.conf", strings.NewReader("nameserver some-internal-ip"), os.FileMode(fs.FileModeRootReadWrite)),
				)

				Expect(cDnsmasq.Run()).To(Succeed())
			})
		})

		Context("when external-resolv.conf already exists", func() {
			It("should not overwrite the file", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("resolvconf", "--disable-updates"),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "stop"),
					mockCmdRunner.EXPECT().Output("ip", "route", "get", "1").Return([]byte("some-ip via some-other-ip dev eth0  src some-internal-ip\n cache"), nil),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/domain", strings.NewReader("address=/.some-domain/some-external-ip\naddress=/.cf.internal/127.0.0.1"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/interface", strings.NewReader("listen-address=some-internal-ip"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.conf", strings.NewReader("resolv-file=/var/pcfdev/external-resolv.conf"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Exists("/var/pcfdev/external-resolv.conf").Return(true, nil),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "start"),
					mockFS.EXPECT().Write("/etc/resolv.conf", strings.NewReader("nameserver some-internal-ip"), os.FileMode(fs.FileModeRootReadWrite)),
				)

				Expect(cDnsmasq.Run()).To(Succeed())
			})
		})

		Context("when there is an error disabling resolvconf updates", func() {
			It("should return the error", func() {
				mockCmdRunner.EXPECT().Run("resolvconf", "--disable-updates").Return(errors.New("some-error"))

				Expect(cDnsmasq.Run()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error stopping dnsmasq", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("resolvconf", "--disable-updates"),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "stop").Return(errors.New("some-error")),
				)

				Expect(cDnsmasq.Run()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error writing the dnsmasq conf", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("resolvconf", "--disable-updates"),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "stop"),
					mockCmdRunner.EXPECT().Output("ip", "route", "get", "1").Return([]byte("some-ip via some-other-ip dev eth0  src some-internal-ip\n cache"), nil),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/domain", strings.NewReader("address=/.some-domain/some-external-ip\naddress=/.cf.internal/127.0.0.1"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/interface", strings.NewReader("listen-address=some-internal-ip"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.conf", strings.NewReader("resolv-file=/var/pcfdev/external-resolv.conf"), os.FileMode(fs.FileModeRootReadWrite)).Return(errors.New("some-error")),
				)

				Expect(cDnsmasq.Run()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error checking the resolv conf", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("resolvconf", "--disable-updates"),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "stop"),
					mockCmdRunner.EXPECT().Output("ip", "route", "get", "1").Return([]byte("some-ip via some-other-ip dev eth0  src some-internal-ip\n cache"), nil),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/domain", strings.NewReader("address=/.some-domain/some-external-ip\naddress=/.cf.internal/127.0.0.1"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/interface", strings.NewReader("listen-address=some-internal-ip"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.conf", strings.NewReader("resolv-file=/var/pcfdev/external-resolv.conf"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Exists("/var/pcfdev/external-resolv.conf").Return(false, errors.New("some-error")),
				)

				Expect(cDnsmasq.Run()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error reading the resolv conf", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("resolvconf", "--disable-updates"),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "stop"),
					mockCmdRunner.EXPECT().Output("ip", "route", "get", "1").Return([]byte("some-ip via some-other-ip dev eth0  src some-internal-ip\n cache"), nil),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/domain", strings.NewReader("address=/.some-domain/some-external-ip\naddress=/.cf.internal/127.0.0.1"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/interface", strings.NewReader("listen-address=some-internal-ip"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.conf", strings.NewReader("resolv-file=/var/pcfdev/external-resolv.conf"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Exists("/var/pcfdev/external-resolv.conf").Return(false, nil),
					mockFS.EXPECT().Read("/etc/resolv.conf").Return(nil, errors.New("some-error")),
				)

				Expect(cDnsmasq.Run()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error writing the pcfdev external resolv.conf", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("resolvconf", "--disable-updates"),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "stop"),
					mockCmdRunner.EXPECT().Output("ip", "route", "get", "1").Return([]byte("some-ip via some-other-ip dev eth0  src some-internal-ip\n cache"), nil),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/domain", strings.NewReader("address=/.some-domain/some-external-ip\naddress=/.cf.internal/127.0.0.1"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/interface", strings.NewReader("listen-address=some-internal-ip"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.conf", strings.NewReader("resolv-file=/var/pcfdev/external-resolv.conf"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Exists("/var/pcfdev/external-resolv.conf").Return(false, nil),
					mockFS.EXPECT().Read("/etc/resolv.conf").Return([]byte("nameserver 127.0.0.1\nnameserver some-external-nameserver\nnameserver some-other-external-nameserver"), nil),
					mockFS.EXPECT().Write("/var/pcfdev/external-resolv.conf", strings.NewReader("nameserver some-external-nameserver\nnameserver some-other-external-nameserver"), os.FileMode(fs.FileModeRootReadWrite)).Return(errors.New("some-error")),
				)

				Expect(cDnsmasq.Run()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error starting dnsmasq", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("resolvconf", "--disable-updates"),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "stop"),
					mockCmdRunner.EXPECT().Output("ip", "route", "get", "1").Return([]byte("some-ip via some-other-ip dev eth0  src some-internal-ip\n cache"), nil),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/domain", strings.NewReader("address=/.some-domain/some-external-ip\naddress=/.cf.internal/127.0.0.1"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/interface", strings.NewReader("listen-address=some-internal-ip"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.conf", strings.NewReader("resolv-file=/var/pcfdev/external-resolv.conf"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Exists("/var/pcfdev/external-resolv.conf").Return(false, nil),
					mockFS.EXPECT().Read("/etc/resolv.conf").Return([]byte("nameserver 127.0.0.1\nnameserver some-external-nameserver\nnameserver some-other-external-nameserver"), nil),
					mockFS.EXPECT().Write("/var/pcfdev/external-resolv.conf", strings.NewReader("nameserver some-external-nameserver\nnameserver some-other-external-nameserver"), os.FileMode(fs.FileModeRootReadWrite)),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "start").Return(errors.New("some-error")),
				)

				Expect(cDnsmasq.Run()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error writing to the /etc/resolv.conf", func() {
			It("should return the error", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("resolvconf", "--disable-updates"),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "stop"),
					mockCmdRunner.EXPECT().Output("ip", "route", "get", "1").Return([]byte("some-ip via some-other-ip dev eth0  src some-internal-ip\n cache"), nil),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/domain", strings.NewReader("address=/.some-domain/some-external-ip\naddress=/.cf.internal/127.0.0.1"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/interface", strings.NewReader("listen-address=some-internal-ip"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.conf", strings.NewReader("resolv-file=/var/pcfdev/external-resolv.conf"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Exists("/var/pcfdev/external-resolv.conf").Return(false, nil),
					mockFS.EXPECT().Read("/etc/resolv.conf").Return([]byte("nameserver 127.0.0.1\nnameserver some-external-nameserver\nnameserver some-other-external-nameserver"), nil),
					mockFS.EXPECT().Write("/var/pcfdev/external-resolv.conf", strings.NewReader("nameserver some-external-nameserver\nnameserver some-other-external-nameserver"), os.FileMode(fs.FileModeRootReadWrite)),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "start"),
					mockFS.EXPECT().Write("/etc/resolv.conf", strings.NewReader("nameserver some-internal-ip"), os.FileMode(fs.FileModeRootReadWrite)).Return(errors.New("some-error")),
				)

				Expect(cDnsmasq.Run()).To(MatchError("some-error"))
			})
		})

		Context("when the internal ip is not returned from the output", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("resolvconf", "--disable-updates"),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "stop"),
					mockCmdRunner.EXPECT().Output("ip", "route", "get", "1").Return([]byte("some-bad-output"), nil),
				)

				Expect(cDnsmasq.Run()).To(MatchError("internal ip could not be parsed from output: some-bad-output"))
			})
		})

		Context("when there is an error retrieving the internal ip", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("resolvconf", "--disable-updates"),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "stop"),
					mockCmdRunner.EXPECT().Output("ip", "route", "get", "1").Return(nil, errors.New("some-error")),
				)

				Expect(cDnsmasq.Run()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error writing the dnsmasq configuration", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("resolvconf", "--disable-updates"),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "stop"),
					mockCmdRunner.EXPECT().Output("ip", "route", "get", "1").Return([]byte("some-ip via some-other-ip dev eth0  src some-internal-ip\n cache"), nil),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/domain", strings.NewReader("address=/.some-domain/some-external-ip\naddress=/.cf.internal/127.0.0.1"), os.FileMode(fs.FileModeRootReadWrite)).Return(errors.New("some-error")),
				)

				Expect(cDnsmasq.Run()).To(MatchError("some-error"))
			})
		})

		Context("when there is an error writing the dnsmasq interface configuration", func() {
			It("should return an error", func() {
				gomock.InOrder(
					mockCmdRunner.EXPECT().Run("resolvconf", "--disable-updates"),
					mockCmdRunner.EXPECT().Run("service", "dnsmasq", "stop"),
					mockCmdRunner.EXPECT().Output("ip", "route", "get", "1").Return([]byte("some-ip via some-other-ip dev eth0  src some-internal-ip\n cache"), nil),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/domain", strings.NewReader("address=/.some-domain/some-external-ip\naddress=/.cf.internal/127.0.0.1"), os.FileMode(fs.FileModeRootReadWrite)),
					mockFS.EXPECT().Write("/etc/dnsmasq.d/interface", strings.NewReader("listen-address=some-internal-ip"), os.FileMode(fs.FileModeRootReadWrite)).Return(errors.New("some-error")),
				)

				Expect(cDnsmasq.Run()).To(MatchError("some-error"))
			})
		})
	})

	Describe("#Distro", func() {
		It("should return 'oss'", func() {
			Expect(cDnsmasq.Distro()).To(Equal(provisioner.DistributionOSS))
		})
	})
})
