package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

func drainPickerFeed(ch <-chan PickerItem, done <-chan struct{}) tea.Cmd {
	return func() tea.Msg {
		batch := make([]PickerItem, 0, pickerFeedBatchSize)
		var flush <-chan time.Time
		for {
			select {
			case item, ok := <-ch:
				if !ok {
					return pickerFeedMsg{items: batch}
				}
				batch = append(batch, item)
				if len(batch) == 1 {
					flush = time.After(pickerFeedFlushWait)
				}
				if len(batch) >= pickerFeedBatchSize {
					return pickerFeedMsg{items: batch, feed: ch, done: done}
				}
			case <-flush:
				return pickerFeedMsg{items: batch, feed: ch, done: done}
			case <-done:
				return pickerFeedMsg{items: batch}
			}
		}
	}
}

func drainDynamicFeed(gen int, ch <-chan PickerItem) tea.Cmd {
	return func() tea.Msg {
		batch := make([]PickerItem, 0, pickerFeedBatchSize)
		var flush <-chan time.Time
		for {
			select {
			case item, ok := <-ch:
				if !ok {
					return pickerDynamicFeedMsg{gen: gen, items: batch}
				}
				batch = append(batch, item)
				if len(batch) == 1 {
					flush = time.After(pickerFeedFlushWait)
				}
				if len(batch) >= pickerFeedBatchSize {
					return pickerDynamicFeedMsg{
						gen: gen, items: batch, feed: ch,
					}
				}
			case <-flush:
				return pickerDynamicFeedMsg{gen: gen, items: batch, feed: ch}
			}
		}
	}
}
