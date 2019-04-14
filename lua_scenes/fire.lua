local scene = {}

scene.meta = {
  name = "Fire",
  description = "Embers ignite and travel."
}

scene.FADED = 0.1
scene.IDLE_TIME = 7
scene.IDLE_ENERGY = 1
scene.TOUCH_ENERGY = 6.5
scene.IDLE_FLOOR = 0.2
scene.RANGE_SCALE = 1.8

local cm
local ceil = math.ceil
local log = math.log
local sqrt = math.sqrt
local floor = math.floor
local random = math.random
local table_remove = table.remove
local max = math.max
local min = math.min

local s
local set

function scene:createScene(event)
  s = self.screen
  set = self.settings

  cm = set.color_multiplier

  self.IDLE_THRESHOLD = (self.IDLE_ENERGY + 1) * cm
  -- 6.8 ~= 0.8 => nearly-red
  self.CM_CAP = cm * 6.8

  self.squares = Squares.new(s, set)
  for i = 1, #self.squares do
    local c = self.squares[i]
    for j = 1, #c do
      local sq = c[j]
      local new_scale = 0.7 + (1 / 6 / 3)
      sq.base_scale = sq.xScale
      sq.xScale = sq.base_scale  * new_scale
      sq.yScale = sq.base_scale  * new_scale
      sq.fade_floor = self.IDLE_FLOOR
      sq.blocked = {}
      sq.energy_hue = cm
    end
  end
  self.range_scale = self.RANGE_SCALE * (self.squares.total_squares / 768)
  self.fade_multiplier = .005
  self.events = 0
end

local plus = {
  { -1, 0 },
  { 1, 0 },
  { 0, -1 },
  { 0, 1 },
}

local diag = {
  { -1, -1 },
  { -1, 1 },
  { 1, -1 },
  { 1, 1 },
}

local function scale(s, x, y)
  return (x * s), (y * s)
end

-- the larger of the two, plus a fraction of the smaller
local function scale_add(a, b, divisor)
  if a > b then
    local t = b
    b = a
    a = t
  end
  local scale = 1 / divisor
  if a >= 1 then
    scale = scale * (a * a) / (b * b)
  else
    scale = scale * a / b
  end
  return b + (a * scale)
end

function scene:spread_plus(square, range)
  if range < 1 then
    return
  end
  local mod = (square.energy - range + 1) / (2 * cm)
  if mod > 5 then
    mod = 5
  end
  local hue_mod = range * cm / self.range_scale
  local hue = square.energy_hue - hue_mod
  -- Util.printf("spreading %d, %d by %d: nrg %.2f/%.2f nrg_hue %.1f, hue %.1f (%.1f)",
    -- square.logical_x, square.logical_y, range, square.vested_energy or -1, square.energy or -1, square.energy_hue or -1, square.hue, hue)
  local effective = square.energy_hue / cm
  if effective > 4 then
    effective = 4
  end
  local alpha = (square.alpha * effective / 4) - (0.1 * range)
  local spread_any = false
  for j = 1, #plus do
    local sq = square:find(scale(range, unpack(plus[j])))
    if sq then
      if not sq.blocked[square] then
	spread_any = self:energize(sq, mod, hue, alpha, square) or spread_any
      end
    end
  end
  if (range % 2) == 0 then
    range = range / 2
    mod = mod / 2
    alpha = alpha / 2
    hue = hue - cm / 2
    for j = 1, #diag do
      local sq = square:find(scale(range, unpack(diag[j])))
      if sq then
	if not sq.blocked[square] then
	  spread_any = self:energize(sq, mod, hue, alpha, square) or spread_any
	end
      end
    end
  end
end

function scene:enterFrame(event)
  local this_frame_fade = self.fade_multiplier + (sqrt(self.events) * .002)
  local removes = {}
  self.events = 0
  for i = 1, #self.squares do
    local column = self.squares[i]
    for idx = 1, #column do
      local square = column[idx]
      local bcount = 0
      local rcount = 0
      if square.energy and square.energy > 0.05 then
	self.events = self.events + 1
	if square.vested_energy < square.energy then
	  local old_range = floor((square.spread_so_far or 0) * self.range_scale / cm)
	  local diff = square.energy - square.vested_energy
	  local scale = min(diff / cm, square.vested_energy)
	  square.vested_energy = min(square.energy, square.vested_energy + scale + 1)
	  if square.spreading then
	    local new_range = floor(square.vested_energy * self.range_scale / cm)
	    for j = old_range + 1, new_range do
	      self:spread_plus(square, j)
	    end
	    square.spread_so_far = square.vested_energy
	  end
	else
	  square.force = false
	  square.blocked = {}
	  square.spreading = false
	  square.energy = max(square.energy * 0.90 - 0.1, 0)
	  square.vested_energy = square.energy
	  if square.energy_hue > cm then
	    square.energy_hue = max(cm, square.energy_hue - (0.5 * (square.energy_hue / cm)))
	  end
	end
	if square.hue ~= square.energy_hue then
	  local diff = square.energy_hue - square.hue
	  if square.hue < square.energy_hue then
	    square.hue = min(square.energy_hue, square.hue + diff / 3 + 2)
	  else
	    square.hue = max(cm, square.hue + diff / 4)
	  end
	  square:colorize()
	end
      else
	square.blocked = {}
	square.energy = nil
	square.vested_energy = 0
	removes[#removes + 1] = i
        if square.fade_floor < self.IDLE_FLOOR then
          square.fade_floor = min(square.fade_floor + 0.001, self.IDLE_FLOOR)
        end
	if square.hue > cm then
	  square.hue = max(square.hue * 0.99 - 1, cm)
	  square:colorize()
	  if square.alpha ~= square.fade_floor then
	    square.alpha = max(square.fade_floor, square.alpha - (this_frame_fade / 4))
	  end
	else
	  if square.alpha ~= square.fade_floor then
	    square.alpha = max(square.fade_floor, square.alpha - this_frame_fade)
	  end
	end
      end
      if random(self.squares.total_squares) == 1 then
	square.blocked = {}
	if square.alpha < 1 and square.hue < cm * 4 then
	  self:energize(square, (2 / cm) + random() + 1, square.hue + cm / 2, .1, { force = true })
	  -- and make this one a little stickier
	  square.energy = square.energy + 3
	end
      end
      local new_scale = 0.7 + ((square.energy or 0) / cm / 6 / 3)
      square.xScale = square.base_scale * new_scale
      square.yScale = square.base_scale * new_scale
    end
  end
  self.cooldown = self.cooldown - 1
  if self.cooldown < 1 then
    self.cooldown = self.IDLE_TIME
    local square = self.squares[random(self.squares.columns)][random(self.squares.rows)]
    local energy = self.IDLE_ENERGY + random(4) + random(3) - 2
    Sounds.playoctave(floor(energy), 0)
    self:energize(square, energy / 2, energy * cm / 2, 1)
    square.spreading = true
    square.spread_so_far = 0
    square.energy = square.energy + 2
    --[[
    if energy > 5 then
      energy = energy - 2
      for i = 1, #diag do
        local sq = square:find(unpack(diag[i]))
        self:energize(sq, energy, energy * cm, 1)
        sq.spreading = true
	sq.spread_so_far = 0
      end
    end
    ]]--
  end
  -- Util.printf("%d events, %d to remove (fade %.1f%%)", self.events, #removes, this_frame_fade * 100)
end

function scene:willEnterScene(event)
  for x, column in ipairs(self.squares) do
    for y, square in ipairs(column) do
      square.hue = cm
      square.alpha = self.FADED
      square:colorize()
    end
  end
  self.tone_offset = 1
  self.tone_base_offset = 0
  self.cooldown = 0
end

function scene:energize(square, amount, hue, newalpha, source)
  amount = amount * cm
  newalpha = max(newalpha or 1, self.IDLE_FLOOR)
  if amount < 0.1 then
    return false
  end
  -- limit amounts added
  if amount > self.CM_CAP then
    amount = self.CM_CAP
  end
  local old_hue = hue or amount or cm
  hue = scale_add(hue or amount or cm, square.energy_hue or 0, 8)
  if hue < cm then
    -- allow hue to go slightly purpler than red.
    hue = cm - 1
  else
    if hue > self.CM_CAP then
      hue = self.CM_CAP
    end
  end
  -- so, do we forcibly update?
  local force_value = true
  if source then
    if not source.force then
      if square.energy and square.energy > amount then
        force_value = false
      end
    else
      -- don't downgrade energy
      hue = max(square.energy or cm, max(hue, square.energy_hue or cm))
      amount = max(amount, square.energy or 0)
    end
  else
    if square.energy and square.energy * 5 > amount then
      force_value = false
    end
  end
  if force_value then
    square.energy = amount
    square.vested_energy = square.vested_energy or 0
    square.spread_so_far = 0
    square.energy_hue = hue
    square.alpha = min(scale_add(square.alpha, newalpha, 1), 1)
    if square.energy > self.IDLE_THRESHOLD + (cm * 2) and hue > (cm * 4) then
      square.spreading = true
    end
    -- Util.printf("new event at %d, %d: energy %.1f, alpha %.2f, hue %.1f",
      -- square.logical_x, square.logical_y, square.energy, square.alpha, square.energy_hue)
  else
    local old_energy = square.energy or 0
    square.energy = scale_add(square.energy, amount, 16)
    square.alpha = max(square.alpha, min(newalpha, square.alpha + amount))
    square.energy_hue = hue
    if square.energy > self.IDLE_THRESHOLD and old_energy < self.IDLE_THRESHOLD then
      square.spreading = true
      square.spread_so_far = 0
    end
    -- Util.printf("energized %d, %d from %.1f to %.1f, hue %.1f, new hue %.1f, alpha %.2f, spreading %s", square.logical_x, square.logical_y, old_energy, square.energy, hue, square.energy_hue, square.alpha, tostring(square.spreading))
  end
  if source then
    square.blocked[source] = true
  end
  -- rarely-hit spots energize a little further
  square.energy = square.energy + square.fade_floor * cm
  square.fade_floor = max(0.001, square.fade_floor - amount / cm / 2)
  return square.spreading
end

function scene:touch_magic(state, ...)
  local t = os.time()
  self.touch_history = self.touch_history or {}
  self.touch_energy = self.touch_energy or {}
  for i, v in pairs(state.points) do
    local square = self.squares:from_screen(v.current)
    if square then
      if not self.touch_energy[i] then
        self.touch_energy[i] = scene.TOUCH_ENERGY
      end
      if (not square.touched) or (t - square.touched > 1) or square.toucher ~= i then
	local energy = self.touch_energy[i]
	if self.touch_history[i] then
	  self.touch_history[i] = self.touch_history[i] + 1
	  energy = energy - sqrt(self.touch_history[i])
	else
	  if not v.done then
	    self.touch_history[i] = 0
	  end
	end
	energy = max(energy, 1.2)
	self.touch_energy[i] = scale_add(self.touch_energy[i], (square.energy or 0) / cm, 2)
	if v.new_event then
	  Sounds.playoctave(min(floor((square.energy_hue or cm) / cm + 0.5), 6), 1)
	end
        self:energize(square, energy)
        square.alpha = 1
        square.spreading = true
	square.force = true
      end
      if v.done then
	if self.touch_history[i] then
	  self.touch_history[i] = nil
          square.touched = t
	  square.toucher = i
        else
          square.touched = false
	  square.toucher = nil
	end
	self.touch_energy[i] = nil
      else
        square.touched = t
      end
    end
  end
end

function scene:enterScene(event)
  self.tone_offset = 1
  self.tone_base_offset = 0
end

function scene:destroyScene(event)
  self.squares:removeSelf()
  self.squares = nil
end

return scene
