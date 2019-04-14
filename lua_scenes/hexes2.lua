local scene = {}

scene.meta = {
  name = "Painting Hexes",
  description = "Hexes wander the screen, trying to convert things to their own color. Touch events can smear colors around."
}

scene.FADED = 0.75
scene.CYCLE = 6

local ceil = math.ceil
local floor = math.floor
local max = math.max
local min = math.min

local s
local set

function scene:createScene(event)
  s = self.screen
  set = self.settings
  self.hexes = Hexes.new(s, set, set.ants, set.color_multiplier)
  self.half_colors = set.total_colors / 2
end

function scene:enterFrame(event)
  local ant = table.remove(self.ants, 1)
  local choices = {
    { dir = Hexes.turn[ant.dir].right },
    { dir = Hexes.turn[ant.dir].ahead },
    { dir = Hexes.turn[ant.dir].left },
  }
  ant.hex.ant = nil
  for i, ch in ipairs(choices) do
    ch.hex = (Hexes.dir[ch.dir])(ant.hex)
    ch.dist = self.hexes.color_dist(ch.hex.hue, ant.hue) + self.half_colors * (1 - ch.hex.alpha) / 2
  end
  table.sort(choices, function(a, b) return a.dist > b.dist end)
  ant.dir = choices[1].dir
  ant.hex = choices[1].hex
  ant.hex.ant = ant
  ant.hex.hue = ant.hue
  ant.hex.alpha = 1
  ant.hex:colorize()
  ant.light:move(ant.hex)

  -- leave a trail!
  local behind
  behind = Hexes.dir[Hexes.turn[ant.dir].hard_right](ant.hex)
  behind.alpha = min(1, behind.alpha + 0.1)
  behind.hue = self.hexes.color_towards(behind.hue, ant.hue)
  if behind.ant then
    self.make_sound = behind.ant.hue / set.color_multiplier
    self.make_octave = 1
  end
  behind:colorize()

  behind = Hexes.dir[Hexes.turn[ant.dir].hard_left](ant.hex)
  behind.alpha = min(1, behind.alpha + 0.1)
  behind.hue = self.hexes.color_towards(behind.hue, ant.hue)
  if behind.ant then
    self.make_sound = behind.ant.hue / set.color_multiplier
    self.make_octave = 1
  end
  behind:colorize()

  table.insert(self.ants, ant)

  local column = self.hexes[self.fade_column]
  for _, hex in ipairs(column) do
    hex.alpha = max(0, hex.alpha - .003)
  end
  self.fade_column = (self.fade_column % #self.hexes) + 1
  local removes = {}
  for k, splash in ipairs(self.splashes) do
    splash.cooldown = splash.cooldown - 1
    if splash.cooldown < 1 then
      self.make_sound = floor(splash.hue / set.color_multiplier + 0.5)
      self.make_octave = splash.octave
      proc = coroutine.create(scene.process_hex)
      splash.cooldown = 2
      splash.hex:splash(1, 1, proc, splash.hue)
      removes[#removes + 1] = k
    end
  end
  local maybe_make_sound = false
  if ant.index % 2 == 1 then
    if self.sound_toggle then
      Sounds.playoctave(3 - ant.index, 0)
    end
    self.sound_toggle = not self.sound_toggle
    maybe_make_sound = true
  end
  if self.make_sound and maybe_make_sound then
    Sounds.playoctave(self.make_sound, self.make_octave)
    self.make_sound = nil
  end
  while #removes > 0 do
    table.remove(self.splashes, table.remove(removes))
  end
end

function scene.process_hex(hex, inc, hue)
  local self = scene
  while hex do
    local old = hex.hue
    hex.hue = hex.hexes.color_towards(hex.hue, hue)
    hex:colorize()
    hex.alpha = min(1, hex.alpha + 0.1)
    hex, increment, hue = coroutine.yield(true)
  end
end

function scene:willEnterScene(event)
  for x, column in ipairs(self.hexes) do
    for y, hex in ipairs(column) do
      hex.hue = math.random(6 * set.color_multiplier)
      hex.alpha = self.FADED
      hex:colorize()
    end
  end
  self.view.alpha = 0
end

local recent_touch = { }

function scene:touch_magic(state, ...)
  -- actually, we care because you might be holding a note.
  -- if state.events == 0 then
  --   return
  -- end
  for idx, event in ipairs(state.points) do
    local idx = event.idx
    recent_touch[idx] = recent_touch[idx] or {}
    local touch = recent_touch[idx]
    if event.events ~= 0 then
      local hit_hexes = {}
      if not touch.hue then
	if not event.start or not event.start.x then
	  Util.printf("start_hex event bogosity:")
	  Util.dump(event)
	end
	local start_hex = self.hexes:from_screen(event.start)
	if start_hex then
	  touch.hue = start_hex.hue
	else
	  touch.hue = 1
	end
      end
      for i, e in ipairs(event.previous) do
	local new = self.hexes:from_screen(e)
	if new and new ~= touch.last_hex then
	  hit_hexes[new] = true
	end
      end
      if event.current and not event.done then
	local hex = self.hexes:from_screen(event.current)
	if hex and hex ~= touch.last_hex then
	  hit_hexes[hex] = true
	end
	touch.last_hex = hex
      end
      for hex, _ in pairs(hit_hexes) do
	table.insert(self.splashes, { cooldown = 1, hex = hex, hue = touch.hue, octave = idx })
      end
      if event.done then
	recent_touch[idx] = nil
      end
    elseif not event.done then
      local hue = touch and touch.hue or 1
      self.make_sound = floor(hue / set.color_multiplier + 0.5)
      self.make_octave = idx
    end
  end
end

function scene:enterScene(event)
  self.splashes = {}
  self.ants = {}
  self.sound_delay = 0
  self.make_sound = nil
  for i, h in ipairs(self.hexes.highlights) do
    local ant =  {
      x = math.random(self.hexes.columns),
      y = math.random(self.hexes.rows),
      index = i,
      light = h,
      dir = Hexes.directions[math.random(#Hexes.directions)],
      hue = i * set.color_multiplier,
    }
    ant.hex = self.hexes:find(ant.x, ant.y)
    ant.hex.hue = ant.hue
    ant.hex:colorize()
    h.hue = ant.hue
    h.alpha = 1
    h:colorize()
    self.ants[i] = ant
  end
  self.fade_column = 1
end

function scene:destroyScene(event)
  self.hexes:removeSelf()
  self.hexes = nil
end

return scene
