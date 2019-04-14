local scene = {}

scene.meta = {
  name = "Cascade",
  description = "Cellular automaton. Touch squares to activate, tilt device to change direction of cascade."
}

scene.FADE_RATE = 6
scene.FADED = 0.75

local max = math.max
local min = math.min

local s
local set

function scene:createScene(event)
  s = self.screen
  set = self.settings
  self.squares = Squares.new(s, set)
  self.total_colors = set.color_multiplier * #Rainbow.hues
end

function scene:enterFrame(event)
  local dir = system.orientation
  -- Util.printf("%s", tostring(dir))
  local step_back = false
  local row_mod, square_mod
  local square_adjust
  if dir == 'portraitUpsideDown' then
    row_mod = self.squares.rows
    square_mod = self.squares.columns
    square_adjust = 0
    self.active_row = (self.active_row - 1) % row_mod + 1
    prev = self.squares.r[self.active_row]
    next = prev[1]:find(0, -1).row
    self.active_row = (self.active_row - 2) % row_mod + 1
  elseif dir == 'landscapeLeft' then
    row_mod = self.squares.columns
    square_mod = self.squares.rows
    square_adjust = 0
    self.active_row = (self.active_row - 1) % row_mod + 1
    prev = self.squares[self.active_row]
    next = prev[1]:find(1, 0).column
    self.active_row = self.active_row % row_mod + 1
  elseif dir == 'landscapeRight' then
    row_mod = self.squares.columns
    square_mod = self.squares.rows
    square_adjust = -2
    self.active_row = (self.active_row - 1) % row_mod + 1
    prev = self.squares[self.active_row]
    next = prev[1]:find(-1, 0).column
    self.active_row = (self.active_row - 2) % row_mod + 1
  else
    -- portrait and anything else
    row_mod = self.squares.rows
    square_mod = self.squares.columns
    square_adjust = -2
    self.active_row = (self.active_row - 1) % row_mod + 1
    prev = self.squares.r[self.active_row]
    next = prev[1]:find(0, 1).row
    self.active_row = (self.active_row % self.squares.rows) + 1
  end
  if self.active_row == 1 then
    step_back = true
  end
  local previous_state = 0
  local toggles = 0
  local all_true = true
  local all_false = true
  -- if each "row" is a column, we want to adjust by fade rate, otherwise
  -- by fade rate times the ratio of column size to row size. I think.
  local fade_ratio = scene.FADE_RATE * #self.squares / row_mod / 2
  for i, square in ipairs(next) do
    local above = prev[i]
    local before = prev[(i + square_adjust) % square_mod + 1]
    if step_back then
      square = next[(i + square_adjust) % square_mod + 1]
    end
    local new = (above.compute + before.compute)
    if square.flag then
      new = new + 1
      square.flag = nil
    end
    new = new % 2
    square.compute = new
    if square.compute ~= previous_state then
      toggles = toggles + 1
      previous_state = square.compute
    end
    if square.compute == 1 then
      all_false = false
      square.alpha = 1
      -- math.min(1, square.alpha + (.022 * fade_ratio))
    else
      all_true = false
      square.alpha = min(1, square.alpha + (.0065 * fade_ratio))
    end
    square.hue = self.colors[square.compute % 2 + 1]
    square:colorize()
  end
  -- turn one light on randomly next time.
  if all_false then
    local square = next[math.random(#next)]
    square.compute = 1
    square.alpha = 1
    square.hue = self.colors[square.compute % 2 + 1]
    square:colorize()
  end
  self.colors[1] = (self.colors[1] % self.total_colors) + 1
  local idx = self.colors[1] % set.color_multiplier
  if idx == 0 or idx == set.color_multiplier / 2 then
    Sounds.playexact(2 * self.colors[1] / set.color_multiplier, 0.8)
  end
  self.colors[2] = (self.colors[2] % self.total_colors) + 1
  for i = 1, scene.FADE_RATE do
    local column = self.squares[self.fade_column]
    for _, square in ipairs(column) do
      square.alpha = max(0.005, square.alpha - .006)
    end
    self.fade_column = (self.fade_column % #self.squares) + 1
  end
  Sounds.play(toggles)
end

function scene:willEnterScene(event)
  for x, column in ipairs(self.squares) do
    for y, square in ipairs(column) do
      square.hue = 1
      square.compute = 0
      square.alpha = self.FADED + (y == 1 and 0.1 or 0.0)
      square:colorize()
    end
  end
  self.active_row = 1
  self.index = 0
  self.colors = { 1, 1 + set.color_multiplier }
  self.squares[1][1].hue = self.colors[2]
  self.squares[1][1]:colorize()
  self.squares[1][1].compute = 1
  self.fade_column = 1
end

function scene:touch_magic(state)
  self.touch_event_states = self.touch_event_states or {}
  if state.events > 0 then
    for i, event in pairs(state.points) do
      if event.events > 0 then
        local hitboxes = {}
	local square
	for i, e in ipairs(event.previous) do
	  square = self.squares:from_screen(e)
	  if square and self.touch_event_states[event] ~= square then
	    hitboxes[square] = true
	  end
	end
	square = self.squares:from_screen(event.current)
	if square then
	  if self.touch_event_states[event] ~= square then
	    hitboxes[square] = true
	  end
	  self.touch_event_states[event] = square
	end
	for square, _ in pairs(hitboxes) do
	  square.flag = true
	  square.alpha = 1
	  square.hue = square.hue + set.color_multiplier
	  square:colorize()
	end
      end
      if event.done then
        self.touch_event_states[event] = nil
      end
    end
  end
end

function scene:enterScene(event)
  self.toward = nil
end

function scene:exitScene(event)
  self.toward = nil
end

function scene:destroyScene(event)
  self.squares:removeSelf()
  self.squares = nil
end

return scene
