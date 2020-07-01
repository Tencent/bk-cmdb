package util_test

import (
	"configcenter/src/common/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)
var _ = Describe("Test SliceInterfaceToString,Int64,Bool", func() {
	Context("Test SliceInterfaceToString ", func() {
		It("number1", func() {
			var input = []interface{}{"abcd","1111",""}
			var shouldout = []string{"abcd","1111",""}
			results ,err := util.SliceInterfaceToString(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(shouldout).To(Equal(results))
		})
		It("number2", func() {
			var input = []interface{}{}
			var shouldout = []string{}
			results ,err := util.SliceInterfaceToString(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(shouldout).To(Equal(results))
		})
		It("shoule error", func() {
			var input = []interface{}{"abcd",12}
			_ ,err := util.SliceInterfaceToString(input)
			Expect(err).To(HaveOccurred())
		})
	})
	Context("Test SliceInterfaceToBool", func() {
		It("number1", func() {
			var input = []interface{}{true,false,true,true}
			var shouldout = []bool{true,false,true,true}
			results ,err := util.SliceInterfaceToBool(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(shouldout).To(Equal(results))
		})
		It("number2", func() {
			var input = []interface{}{}
			var shouldout = []bool{}
			results ,err := util.SliceInterfaceToBool(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(shouldout).To(Equal(results))
		})
		It("shoule error", func() {
			var input = []interface{}{"abcd",12}
			_ ,err := util.SliceInterfaceToBool(input)
			Expect(err).To(HaveOccurred())
		})
	})
	Context("Test SliceInterfaceToInt64", func() {
		It("number1", func() {
			var input = []interface{}{int64(1),int32(32),int(32),uint8(100)}
			var shouldout = []int64{1,32,32,100}
			results ,err := util.SliceInterfaceToInt64(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(shouldout).To(Equal(results))
		})
		It("number2", func() {
			var input = []interface{}{int64(1),int64(32),int64(32),int64(100)}
			var shouldout = []int64{1,32,32,100}
			results ,err := util.SliceInterfaceToInt64(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(shouldout).To(Equal(results))
		})
		It("number3", func() {
			var input = []interface{}{}
			var shouldout = []int64{}
			results ,err := util.SliceInterfaceToInt64(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(shouldout).To(Equal(results))
		})
		It("shoule error", func() {
			var input = []interface{}{"abcd",12}
			_ ,err := util.SliceInterfaceToInt64(input)
			Expect(err).To(HaveOccurred())
		})
	})
})
