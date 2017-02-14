// Package recurring contains more implementations for recurring.R.
package recurring

import (
  "github.com/keep94/gofunctional3/functional"
  "github.com/keep94/sunrise"
  "math"
  "math/rand"
  tasks_recurring "github.com/keep94/tasks/recurring"
  "time"
)

// EachSunset returns the sunsets for a given latitude and longitude.
// lat is the latitude where north is positive and south is negative.
// lon is the longitude where east is positive and west is negative.
func EachSunset(lat, lon float64) tasks_recurring.R {
  return tasks_recurring.RFunc(func(t time.Time) functional.Stream {
    var s sunsetIterator
    s.Around(lat, lon, t)
    for !s.Sunset().After(t) {
      s.AddDays(1)
    }
    return &s
  })
}

// OnOrBefore ensures that the times in r happen on or before
// hour:min. If a time is after hour:min, it is moved earlier to be
// hour:min. If a time is 12 hours or more after hour:min, then it is
// considered to be before hour:min on the next day, and no adjustment is
// made.
func OnOrBefore(r tasks_recurring.R, hour, min int) tasks_recurring.R {
  return tasks_recurring.RFunc(func(t time.Time) functional.Stream {
    s := r.ForTime(t)
    return functional.DropWhile(
        functional.NewFilterer(func(ptr interface{}) error {
          p := ptr.(*time.Time)
          if p.After(t) {
            return functional.Skipped
          }
          return nil
        }),
        &happensBefore{
            Stream: s, hour: hour, min: min, hm: toHourMinute(hour, min)})
  })
}

// Random returns random times at least min apart. The expected difference
// between the times is min + expectedAddition.
func Random(min, expectedAddition time.Duration) tasks_recurring.R {
  return tasks_recurring.RFunc(func(t time.Time) functional.Stream {
    return &random{
        current: t,
        min: min,
        expectedAddition: expectedAddition,
    }
  })
}

type sunsetIterator struct {
  sunrise.Sunrise
}

func (s *sunsetIterator) Next(ptr interface{}) error {
  p := ptr.(*time.Time)
  *p = s.Sunset()
  s.AddDays(1)
  return nil
}

func (s *sunsetIterator) Close() error {
  return nil
}

type random struct {
  current time.Time
  min time.Duration
  expectedAddition time.Duration
}

func (r *random) Next(ptr interface{}) error {
  r.current = r.current.Add(r.min)
  r.current = r.current.Add(
      time.Duration(
          -1.0 * math.Log(1.0 - rand.Float64()) * float64(r.expectedAddition)))
  p := ptr.(*time.Time)
  *p = r.current
  return nil
}

func (r *random) Close() error {
  return nil
}

type happensBefore struct {
  functional.Stream
  hour int
  min int
  hm int
  last time.Time
  started bool
}

func (h *happensBefore) Next(ptr interface{}) (err error) {
  var incoming time.Time
  err = h.Stream.Next(&incoming)
  for ; err == nil; err = h.Stream.Next(&incoming) {
    incoming = h.adjust(incoming)
    if h.started && incoming == h.last {
      continue
    }
    *ptr.(*time.Time) = incoming
    h.last = incoming
    h.started = true
    return
  }
  return
}

func (h *happensBefore) adjust(t time.Time) time.Time {
  hm := toHourMinute(t.Hour(), t.Minute())
  if hm >= h.hm && hm < h.hm + 720 {
    return time.Date(t.Year(), t.Month(), t.Day(), h.hour, h.min, 0, 0, t.Location())
  }
  if hm >= h.hm - 1439 && hm < h.hm - 720 {
    result := time.Date(t.Year(), t.Month(), t.Day(), h.hour, h.min, 0, 0, t.Location())
    return result.AddDate(0, 0, -1)
  }
  return t
}

func toHourMinute(hour, min int) int {
  return 60 * hour + min
}

func init() {
  rand.Seed(time.Now().UnixNano())
}
