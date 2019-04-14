local scene = {}

scene.meta = {
  name = "Spiraling Shape",
  description = "The arms will drift towards touch events."
}

local pi = math.pi
local fmod = math.fmod
local sin = math.sin
local min = math.min
local cos = math.cos
local floor = math.floor
local ceil = math.ceil
local abs = math.abs
local find_line = Util.line
local line_new = Line.new

-- settings and the Screen object
local set
local s

scene.segment_fudge = 5

local rfuncs
local colorfor
local colorize
local color_scale

function scene:createScene(event)
  set = self.settings
  s = self.screen

  self.total_colors = #Rainbow.hues * set.color_multiplier
  rfuncs = Rainbow.funcs_for(set.color_multiplier)
  colorfor = rfuncs.smooth
  colorize = rfuncs.smoothobj

  self.line_segments = self.total_colors
  self.segments_triangle = (self.line_segments * self.line_segments + self.line_segments) / 2
  self.segments_triangle = self.segments_triangle + (self.line_segments * self.segment_fudge)
  color_scale = floor(self.total_colors / set.points)

  self.ids = self.ids or {}
  self.sorted_ids = self.sorted_ids or {}
  self.toward = self.toward or {}

  self.center = Vector.random(s, set, 5)
  self.center.ripples = {}
  self.center.x = s.center.x
  self.center.y = s.center.y
  self.lines = {}
  -- so clicks have something to land on
  self.bg = display.newRect(s, 0, 0, s.size.x, s.size.y)
  self.bg:setFillColor(0, 0)
  s:insert(self.bg)
end

local ripple_pattern = { -1, -2, 0, 2, 1, 0, -1, 0, 1 }

function scene:spiral_from(vec, points, segments)
  local params = find_line(self.center, vec)
  params.theta = params.theta + fmod(vec.theta, pi * 2)
  local rip = {}
  local remove = {}
  for idx, r in ipairs(vec.ripples) do
    if r > 1 then
      for i, v in ipairs(ripple_pattern) do
        rip[r + i] = (rip[r + i] or 0) + v
      end
      vec.ripples[idx] = r - 2
    else
      remove[#remove + 1] = idx
    end
  end
  while #remove > 0 do
    table.remove(vec.ripples, table.remove(remove))
  end
  -- no point in doing the 0/N cases, because they're trivial
  local counter = segments
  for i = 1, segments - 1 do
    local scale = i / segments
    counter = counter + (segments - i) + self.segment_fudge
    local theta = ((counter / self.segments_triangle) * vec.theta) + params.theta
    local r = params.len * scale
    if rip[i] then
      r = r * (1 + 0.03 * rip[i])
    end
    points[i + 1] = points[i + 1] or {}
    points[i + 1].x = r * cos(theta) + self.center.x
    points[i + 1].y = r * sin(theta) + self.center.y
  end
end

function scene:all_lines(color, g)
  if not color then
    color = self.next_color or 1
    self.next_color = (color % self.total_colors) + 1
  end
  g = g or display.newGroup()
  g.sublines = g.sublines or {}
  for i = 1, set.points do
    if not g.sublines[i] then
      g.sublines[i] = self:line(color + i * color_scale, g.sublines[i], i)
      g:insert(g.sublines[i])
    else
      self:line(color + i * color_scale, g.sublines[i], i)
    end
  end
  return g
end

function scene:line(color, g, index)
  g = g or display.newGroup()
  g.points = g.points or {}
  g.segments = g.segments or {}
  self:spiral_from(self.vecs[index], g.points, self.line_segments)
  g.points[1] = self.center
  g.points[self.line_segments + 1] = { x = self.vecs[index].x, y = self.vecs[index].y }
  if #g.segments == self.line_segments then
    for i, seg in ipairs(g.segments) do
      seg:setPoints(g.points[i], g.points[i + 1])
      colorize(seg, color)
      seg:redraw()
      color = color + 1
    end
  else
    for i = 1, self.line_segments do
      local seg = g.segments[i]
      local point = g.points[i]
      local next = g.points[i + 1]
      if seg then
	seg:setPoints(point, next)
	colorize(seg, color)
      else
	local l = line_new(point, next, set.line_depth, colorfor(color))
	l:setThickness(set.line_thickness)
	seg = l
	g.segments[i] = l
	g:insert(l)
      end
      seg:redraw()
      color = color + 1
    end
  end
  return g
end

local vec_add = Util.vec_add
local vec_scale = Util.vec_scale

function scene:move()
  local bounce = false
  for i, v in ipairs(self.vecs) do
    if v:move(self.toward[i]) then
      table.insert(v.ripples, self.line_segments)
      bounce = i
    end
  end
  -- not used during startup
  if bounce and self.next_color then
    Sounds.playoctave(ceil(self.next_color / set.color_multiplier) + bounce * color_scale, bounce)
  end
end

function scene:enterFrame(event)
  local last = table.remove(self.lines, 1)
  for i, l in ipairs(self.lines) do
    l.alpha = i / set.history
  end
  table.insert(self.lines, self:all_lines(nil, last))
  self.lines[#self.lines].alpha = 1
  self:move()
end

function scene:enterScene(event)
  self.lines = {}
  self.next_color = nil
  self.vecs = {}
  for i = 1, set.points do
    self.vecs[i] = Vector.random(s, set)
    self.vecs[i].ripples = {}
    self.vecs[i].theta = 5 * pi
  end
  self:move()
  for i = 1, set.history do
    local g = self:all_lines(i, nil)
    self.view:insert(g)
    g.alpha = i / set.history
    table.insert(self.lines, g)
    self:move()
  end
  self.next_color = 1
end

function scene:touch_magic(state, ...)
  self.toward = {}
  for i, v in pairs(state.points) do
    if not v.done then
      self.toward[i] = v.current
    end
  end
end

function scene:exitScene(event)
  self.sorted_ids = {}
  self.toward = {}
  self.view.alpha = 0
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
