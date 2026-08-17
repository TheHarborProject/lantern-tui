package main

type RunsController struct {
	runs    []AuditWireDTO
	list    *ListPanel
	details *TextPanel
}

func NewRunsController(runs []AuditWireDTO, list *ListPanel, details *TextPanel) *RunsController {
	c := &RunsController{runs: runs, list: list, details: details}
	list.SetEmptyMessage("No audit runs loaded.")
	list.SetOnSelectionChanged(c.selectRun)
	c.refresh()
	return c
}

func (c *RunsController) refresh() {
	items := make([]ListItem[any], 0, len(c.runs))
	for i := range c.runs {
		run := &c.runs[i]
		items = append(items, ListItem[any]{DisplayText: formatRunRow(*run), Object: run})
	}
	c.list.SetItems(items)
	if len(c.runs) == 0 {
		c.details.SetContent("No audit runs loaded.\n")
		return
	}
	c.selectRun(c.list.GetSelectedIndex())
}

func (c *RunsController) selectRun(index int) {
	if index < 0 || index >= len(c.runs) {
		c.details.SetContent("No run selected.\n")
		return
	}
	c.details.SetContent(formatRunDetails(c.runs[index]))
}
