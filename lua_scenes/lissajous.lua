local scene = {}

scene.meta = {
  name = "Lissajous Figures",
  description = "Lissajous curves, with alpha and beta controlled by touch."
}

scene.INSET = 4

local rfuncs
local colorfor
local colorize

local pi = math.pi
local ceil = math.ceil
local twopi = pi * 2
local sin = math.sin
local cos = math.cos
local sqrt = math.sqrt
local min = math.min
local max = math.max
local abs = math.abs
local fmod = math.fmod

local s
local set

function scene:createScene(event)
  s = self.screen
  set = self.settings

  self.total_lines = #Rainbow.hues * set.color_multiplier
  self.total_points = self.total_lines + 1

  rfuncs = Rainbow.funcs_for(set.color_multiplier)
  colorfor = rfuncs.smooth
  colorize = rfuncs.smoothobj

  self.ids = self.ids or {}
  self.sorted_ids = self.sorted_ids or {}
  self.toward = self.toward or {}

  self.x_scale = s.size.x / 2 - (self.INSET / 2)
  self.y_scale = s.size.y / 2 - (self.INSET / 2)

  self.x_offset = self.x_scale + (self.INSET / 2)
  self.y_offset = self.y_scale + (self.INSET / 2)

  self.lines = {}
  self.a = 1
  self.b = 1
  self.delta = 1
  self.line_scale = twopi / self.total_lines
  -- so clicks have something to land on
  self.bg = display.newRect(s, 0, 0, s.size.x, s.size.y)
  self.bg:setFillColor(0, 0)
  s:insert(self.bg)
end

function scene:line(color, g)
  if not color then
    color = self.next_color or 1
  end
  g = g or display.newGroup()
  g.segments = g.segments or {}
  if #g.segments == self.total_lines then
    for i, seg in ipairs(g.segments) do
      seg:setPoints(self.vecs[i], self.vecs[i + 1])
      colorize(seg, color)
      seg:redraw()
      color = color + 1
    end
  else
    for i = 1, self.total_lines do
      local seg = g.segments[i]
      local point = self.vecs[i]
      local next = self.vecs[i + 1]
      if seg then
	seg:setPoints(point, next)
	colorize(seg, color)
      else
	local l = Line.new(point, next, set.line_depth, colorfor(color))
	l:setThickness(set.line_thickness)
	seg = l
	g.segments[i] = l
	g:insert(l)
      end
      seg:redraw()
      color = color + 1
    end
  end
  self.next_color = (color % self.total_lines) + 1
  return g
end

function scene:calc(quiet)
  self.vecs = self.vecs or {}
  if self.target_a and self.a ~= self.target_a then
    if self.a < self.target_a then
      self.a = min(self.target_a, self.a + .06250 * self.scale_delta_a)
    else
      self.a = max(self.target_a, self.a - .06250 * self.scale_delta_a)
    end
    if self.a == self.target_a then
      self.target_a = nil
    end
  end
  if self.target_b and self.b ~= self.target_b then
    if self.b < self.target_b then
      self.b = min(self.target_b, self.b + .06250 * self.scale_delta_b)
    else
      self.b = max(self.target_b, self.b - .06250 * self.scale_delta_b)
    end
    if self.b == self.target_b then
      self.target_b = nil
    end
  end
  self.sign_x = self.sign_x or {}
  self.sign_y = self.sign_y or {}
  self.sound_cooldown = self.sound_cooldown or 0
  local play_sound = false
  for i = 1, self.total_points do
    local t = i * self.line_scale
    local x = sin(self.a * t + self.delta)
    local y = sin(self.b * t - self.delta)
    if not quiet and i % set.color_multiplier == 0 then
      local new_sign_x = x < 0
      local new_sign_y = y < 0
      if new_sign_x ~= self.sign_x[i] or new_sign_y ~= self.sign_y[i] then
        play_sound = true
      end
      self.sign_x[i] = new_sign_x
      self.sign_y[i] = new_sign_y
    end
    x = x * self.x_scale + self.x_offset
    y = y * self.y_scale + self.y_offset
    self.vecs[i] = self.vecs[i] or {}
    self.vecs[i].x = x
    self.vecs[i].y = y
  end
  local delta_scale = max(max(abs(self.b), abs(self.a)), 1)
  self.delta = self.delta + set.delta_delta / delta_scale
  if self.delta > twopi then
    self.delta = self.delta - twopi
  end
  return play_sound
end

function scene:enterFrame(event)
  local last = table.remove(self.lines, 1)
  local play_sound
  for i, l in ipairs(self.lines) do
    l.alpha = sqrt(i / set.history)
  end
  play_sound = self:calc()
  if play_sound and self.sound_cooldown < 1 then
    Sounds.play(ceil(self.next_color / set.color_multiplier))
    self.sound_cooldown = set.sound_delay
  else
    self.sound_cooldown = self.sound_cooldown - (event.actual_frames or 1)
  end
  table.insert(self.lines, self:line(nil, last))
  self.lines[#self.lines].alpha = 1
end

function scene:enterScene(event)
  self.lines = {}
  self.next_color = nil
  self.a = 2
  self.b = 3
  self.target_a = 2
  self.target_b = 3
  self.scale_delta_a = 1
  self.scale_delta_b = 1
  self:calc(true)
  for i = 1, set.history do
    local l = self:line(i, nil)
    l.alpha = sqrt(i / set.history)
    table.insert(self.lines, l)
    s:insert(l)
    l.y = 0
    self:calc(true)
  end
  self.next_color = 1
end

function scene:touch_magic(state)
  local point
  local lowest
  if state.events > 0 then
    for i, v in pairs(state.points) do
      if v.events > 0 then
	if not lowest or i < lowest then
	  lowest = i
	  point = v
	end
      end
    end
  end
  if not point or not point.current then
    return
  end
  local x = point.current.x - s.origin.x
  local y = point.current.y - s.origin.y
  -- Util.printf("lissajous moving to relative %d, %d", x, y)
  local ta = (ceil(x * 8 / s.size.x) - 4) / 2
  local tb = ceil(y * 8 / s.size.y) / 2 + 1
  -- local origa, origb = ta, tb
  local sign_a = ta < 0 and -1 or 1
  if abs(ta) < 1 then
    ta = sign_a
  end
  -- avoid degenerate cases
  local integer_a = fmod(ta, 1) == 0 or fmod(ta, tb) == 0
  local integer_b = fmod(tb, 1) == 0 or fmod(tb, ta) == 0
  local multiples = (fmod(ta, tb) == 0 or fmod(tb, ta) == 0)
  -- if either is a multiple of the other, and neither is 1 exactly,
  -- we'll get redraw/overlap which looks lame
  if multiples and (abs(ta) > 1 and tb > 1) then
    if integer_a then
      ta = ta + 0.5 * sign_a
    else
      tb = tb + 0.5
    end
  end
  self.scale_delta_a = max(1, ta - self.a)
  self.scale_delta_b = max(1, tb - self.b)
  self.target_a = ta
  self.target_b = tb
end

function scene:exitScene(event)
  self.sorted_ids = {}
  self.toward = {}
  for i, l in ipairs(self.lines) do
    l:removeSelf()
  end
  self.lines = {}
end

function scene:destroyScene(event)
  self.bg = nil
  self.lines = nil
end

return scene
