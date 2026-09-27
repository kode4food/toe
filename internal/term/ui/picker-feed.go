package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

type (
	pickerFeedMsg struct {
		picker *Picker
		items  []*PickerItem
		feed   <-chan *PickerItem
		done   <-chan struct{}
	}

	pickerDynamicFeedMsg struct {
		picker *Picker
		gen    int
		items  []*PickerItem
		feed   <-chan *PickerItem
	}
)

const (
	pickerFeedBatchSize = 256
	pickerFeedFlushWait = 40 * time.Millisecond
)

func (p *Picker) drainFeed(
	ch <-chan *PickerItem, done <-chan struct{},
) tea.Cmd {
	return func() tea.Msg {
		batch := make([]*PickerItem, 0, pickerFeedBatchSize)
		var flush <-chan time.Time
		for {
			select {
			case item, ok := <-ch:
				if !ok {
					return pickerFeedMsg{picker: p, items: batch}
				}
				batch = append(batch, item)
				if len(batch) == 1 {
					flush = time.After(pickerFeedFlushWait)
				}
				if len(batch) >= pickerFeedBatchSize {
					return pickerFeedMsg{
						picker: p,
						items:  batch,
						feed:   ch,
						done:   done,
					}
				}
			case <-flush:
				return pickerFeedMsg{
					picker: p,
					items:  batch,
					feed:   ch,
					done:   done,
				}
			case <-done:
				return pickerFeedMsg{picker: p, items: batch}
			}
		}
	}
}

func (p *Picker) drainDynamicFeed(gen int, ch <-chan *PickerItem) tea.Cmd {
	return func() tea.Msg {
		batch := make([]*PickerItem, 0, pickerFeedBatchSize)
		var flush <-chan time.Time
		for {
			select {
			case item, ok := <-ch:
				if !ok {
					return pickerDynamicFeedMsg{
						picker: p,
						gen:    gen,
						items:  batch,
					}
				}
				batch = append(batch, item)
				if len(batch) == 1 {
					flush = time.After(pickerFeedFlushWait)
				}
				if len(batch) >= pickerFeedBatchSize {
					return pickerDynamicFeedMsg{
						picker: p,
						gen:    gen,
						items:  batch,
						feed:   ch,
					}
				}
			case <-flush:
				return pickerDynamicFeedMsg{
					picker: p,
					gen:    gen,
					items:  batch,
					feed:   ch,
				}
			}
		}
	}
}

func (p *Picker) handleFeed(msg pickerFeedMsg) tea.Cmd {
	p.addItems(msg.items)
	if msg.feed != nil {
		return p.drainFeed(msg.feed, msg.done)
	}
	p.finishLoad()
	return nil
}

func (p *Picker) handleDynamicTrigger(msg pickerDynamicTriggerMsg) tea.Cmd {
	if msg.gen != p.load.dynamicGen {
		return nil
	}
	src, ok := p.source.(DynamicPickerSource)
	if !ok {
		return nil
	}
	src.Search(msg.query)
	load := src.Load()
	items := load.Items
	p.load.dynamicStop = load.Stop
	p.list.items = items
	p.list.matched = make([]pickerMatch, len(items))
	for i, item := range items {
		p.list.matched[i] = pickerMatch{item: item}
	}
	if load.Feed != nil {
		return p.drainDynamicFeed(msg.gen, load.Feed)
	}
	p.load.loading = false
	return nil
}

func (p *Picker) handleDynamicFeed(msg pickerDynamicFeedMsg) tea.Cmd {
	if msg.gen != p.load.dynamicGen {
		return nil
	}
	p.addDynamicItems(msg.items)
	if msg.feed != nil {
		return p.drainDynamicFeed(msg.gen, msg.feed)
	}
	p.load.loading = false
	return nil
}

func handlePickerMessage(msg tea.Msg) (*Picker, tea.Cmd) {
	switch msg := msg.(type) {
	case pickerFeedMsg:
		return msg.picker, msg.picker.handleFeed(msg)
	case pickerDynamicTriggerMsg:
		return msg.picker, msg.picker.handleDynamicTrigger(msg)
	case pickerDynamicFeedMsg:
		return msg.picker, msg.picker.handleDynamicFeed(msg)
	case pickerRefreshMsg:
		if msg.gen == msg.picker.load.refreshGen {
			return msg.picker, msg.picker.flushFileChanges()
		}
	}
	return nil, nil
}
