package cache

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Map Cache", func() {
	var (
		c                        UpdatingExpiringCache
		key, val, nonExistentKey string = "test-key", "test-val", "non-existent"
	)

	BeforeEach(func() {
		c = NewMapCache()
	})

	Context("Store", func() {
		It("should store a value", func() {
			Expect(c.Store(key, val)).ToNot(HaveOccurred(), "failed storing a value")
			v, err := c.Get(key)
			Expect(v).To(Equal(val),
				fmt.Sprintf("stored value is incorrect for key %v", key))
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Get", func() {
		BeforeEach(func() {
			c.Store(key, val)
		})

		It("should return a stored value", func() {
			val, err := c.Get(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(val),
				"the returned value is not equal to the stored value")
		})

		It("should return an error when attempting to get a non-existent value", func() {
			_, err := c.Get(key)
			Expect(err).ToNot(HaveOccurred(),
				"an error was not returned when attempting to get a non-existent value")
		})

		It("should not remove a value after it was fethced", func() {
			_, err := c.Get(key)
			Expect(err).ToNot(HaveOccurred())
			_, err = c.Get(key)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Remove", func() {
		BeforeEach(func() {
			c.Store(key, val)
		})

		It("should remove a value", func() {
			Expect(c.Remove(key)).ToNot(HaveOccurred(), "value was not removed")
		})

		It("should store a value after it was removed", func() {
			Expect(c.Remove(key)).ToNot(HaveOccurred(), "value was not removed")
			Expect(c.Store(key, val)).NotTo(HaveOccurred(),
				"value could not be added eventhough it's key is available")
		})

		It("should return an error when attempting to remove a non-existent value", func() {
			Expect(c.Remove("non-existent")).To(HaveOccurred(),
				"an error was not returned when attempting to remove a non-existent value")
		})
	})

	Context("Replace", func() {
		BeforeEach(func() {
			c.Store(key, val)
		})

		It("should replace a permanent value", func() {
			newVal := "new"
			Expect(c.Replace(key, newVal)).ToNot(HaveOccurred())

			val, err := c.Get(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal(newVal), "a value was not replaced")
		})

		It("should replace a temporary value with a permanent one", func() {
			key, val := "tempKey", "tempVal"
			ttl := 3 * time.Second
			Expect(c.StoreWithExpiration(key, val, ttl)).ToNot(HaveOccurred())
			Expect(c.Replace(key, val+val)).ToNot(HaveOccurred())
			Eventually(func() bool {
				retVal, err := c.Get(key)
				return err == nil && retVal == val+val
			}, testTimeout).Should(BeTrue())
		})

		It("should fail to replace a non-existent value", func() {
			Expect(c.Replace(nonExistentKey, nonExistentKey)).To(HaveOccurred())
		})
	})

	Context("Clear", func() {
		It("should remove all values", func() {
			Expect(c.Clear()).ToNot(HaveOccurred())
			_, err := c.Get(key)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Keys", func() {
		BeforeEach(func() {
			Expect(c.Store(key, val)).ToNot(HaveOccurred())
		})

		It("should return a slice of cache keys", func() {
			keys, err := c.Keys()
			Expect(err).ToNot(HaveOccurred())
			Expect(keys).To(HaveLen(1))
			Expect(keys[0]).To(Equal(key))
		})
	})

	Context("StoreWithExpiration", func() {
		It("should add a value", func() {
			c.StoreWithExpiration(key, val, time.Minute)
			Expect(c.Get(key)).To(Equal(val), "value was not stored in cache")
		})

		It("should remove a value after timeout", func() {
			timeout := time.Second * 3
			c.StoreWithExpiration(key, val, timeout)

			Eventually(func() bool {
				_, err := c.Get(key)
				return IsDoesNotExist(err)
			}, testTimeout).Should(BeTrue(),
				"value was not removed from cache after timeout")
		})

		It("should return an error when attempting to override a value", func() {
			c.Store(key, val)
			Expect(c.Store(key, "new-val")).To(HaveOccurred(),
				"expected an error when attempting to override a value")
			storedVal, err := c.Get(key)
			Expect(err).ToNot(HaveOccurred())
			Expect(storedVal).To(Equal(val), "a value was overriden")
		})

		It("should return an error if ttl is non-positive", func() {
			Expect(IsNonPositivePeriod(c.StoreWithExpiration(key, val, 0))).To(BeTrue())
		})
	})

	Context("ReplaceWithExpiration", func() {
		BeforeEach(func() {
			c.Store(key, val)
		})

		It("should replace a permanent value with a temporary one", func() {
			Expect(c.ReplaceWithExpiration(key, val, 3*time.Second)).ToNot(HaveOccurred())
			_, err := c.Get(key)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				_, err := c.Get(key)
				return IsDoesNotExist(err)
			}, testTimeout).Should(BeTrue())
		})
	})

	Context("Expire", func() {
		BeforeEach(func() {
			Expect(c.Store(key, val)).ToNot(HaveOccurred())
		})

		It("should be removed after expiration time is set", func() {
			c.Expire(key, 3*time.Second)

			Eventually(func() bool {
				_, err := c.Get(key)
				return IsDoesNotExist(err)
			}, testTimeout).Should(BeTrue(),
				"value was not removed when expiration time was set")
		})

		It("should return an error when attempting to expire a non-existent key", func() {
			err := c.Expire(nonExistentKey, time.Second)
			Expect(err).To(HaveOccurred(),
				"expected an error when expiring a non-existent key")
		})

		It("should return an error when ttl is non-positive", func() {
			Expect(IsNonPositivePeriod(c.Expire(key, 0))).To(BeTrue())
		})

		It("should update duration of a value", func() {
			newKey, newVal := "newKey", "newVal"
			Expect(c.StoreWithExpiration(newKey, newVal, 5*time.Second)).ToNot(HaveOccurred())
			Expect(c.Expire(newKey, 20*time.Second)).ToNot(HaveOccurred())

			Consistently(func() bool {
				_, err := c.Get(key)
				return err == nil
			}, 18*time.Second).Should(BeTrue())

			Eventually(func() bool {
				_, err := c.Get(newKey)
				return IsDoesNotExist(err)
			}, testTimeout).Should(BeTrue())
		})
	})

	Context("StoreWithUpdate", func() {
		It("should continuosly update the value after the specified duration", func() {
			c.StoreWithUpdate(key, 0, func(currValue interface{}) interface{} {
				return currValue.(int) + 1
			}, 1*time.Second)

			Consistently(func() bool {
				currValue, err := c.Get(key)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					val, err := c.Get(key)
					Expect(err).ToNot(HaveOccurred())
					return val.(int) > currValue.(int)
				}, testTimeout, 500*time.Millisecond).Should(BeTrue(),
					"value should have been updated")

				return true
			}, 10*time.Second).Should(BeTrue(), "value should have been updated")
		})

		It("should return an error if nil updateFunc was provided", func() {
			Expect(IsNilUpdateFunc(c.StoreWithUpdate(key, val, nil, 0))).To(Equal(true))
		})
	})

	Context("ReplaceWithUpdate", func() {
		It("should replace and continously update a permanent value", func() {
			Expect(c.Store(key, val)).ToNot(HaveOccurred())
			updateFunc := func(currValue interface{}) interface{} {
				intVal := currValue.(testStruct).Int
				return testStruct{"Test", intVal + 1}
			}

			Expect(c.ReplaceWithUpdate(key,
				testStruct{"", 0},
				updateFunc, 3*time.Second)).ToNot(HaveOccurred())

			Consistently(func() bool {
				currVal, err := c.Get(key)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					v, err := c.Get(key)
					Expect(err).ToNot(HaveOccurred())
					return v.(testStruct).Int > currVal.(testStruct).Int
				}, testTimeout).Should(BeTrue(),
					"value should have been updated")

				return true
			}, 6*time.Second).Should(BeTrue(),
				"value should have been updating consistently")
		})
	})
})
