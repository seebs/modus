local scene = {}

scene.meta = {
  name = "Bouncing Spline",
  description = "The control points for the spline bounce around the screen or follow touch events."
}

scene.points = 4

local rfuncs
local colorfor
local colorize

local midpoint = Util.midpoint
local partway = Util.partway
local ceil = math.ceil
local floor = math.floor
local sqrt = math.sqrt

local s
local set

function scene:createScene(event)
  s = self.screen
  set = self.settings

  self.total_colors = #Rainbow.hues * set.color_multiplier
  self.color_skip = floor(self.total_colors / (set.history + 2))
  rfuncs = Rainbow.funcs_for(set.color_multiplier)
  colorfor = rfuncs.smooth
  colorize = rfuncs.smoothobj

  self.ids = self.ids or {}
  self.sorted_ids = self.sorted_ids or {}
  self.toward = self.toward or {}

  self.lines = {}

  -- so clicks have something to land on

  self.bg = display.newRect(s, 0, 0, s.size.x, s.size.y)
  self.bg:setFillColor(0, 0)
  s:insert(self.bg)
end

function scene:spline(vecs, points, lo, hi)
  if lo == hi then
    return
  end
  local mid = ceil((lo + hi) / 2)
  local midpt = midpoint(vecs[2], vecs[3])
  local p = points[mid + 1] or {}
  p.x, p.y = midpt.x, midpt.y
  points[mid + 1] = p
  if mid == lo or mid == hi then
    return
  end
  local newvecs = {
    vecs[1],
    midpoint(vecs[1], vecs[2]),
    partway(vecs[2], vecs[3], .25),
    midpt,
    partway(vecs[2], vecs[3], .75),
    midpoint(vecs[3], vecs[4]),
    vecs[4]
  }
  -- if mid == lo+1, then this would just overwrite it
  if mid ~= lo + 1 then
    self:spline(newvecs, points, lo, mid)
  end
  if mid ~= hi - 1 then
    self:spline({ newvecs[4], newvecs[5], newvecs[6], newvecs[7] }, points, mid, hi)
  end
end

function scene:line(color, g)
  if not color then
    color = self.next_color or 1
    self.next_color = ((color + self.color_skip) % self.total_colors) + 1
  end
  g = g or display.newGroup()
  g.points = g.points or {}
  g.segments = g.segments or {}
  self:spline(self.vecs, g.points, 0, self.total_colors)
  g.points[1] = { x = self.vecs[1].x, y = self.vecs[1].y }
  g.points[self.total_colors + 1] = { x = self.vecs[4].x, y = self.vecs[4].y }
  if #g.segments == self.line_segments then
    for i, seg in ipairs(g.segments) do
      seg:setPoints(g.points[i], g.points[i + 1])
      colorize(seg, color)
      seg:redraw()
      color = color + 1
    end
  else
    for i = 1, self.total_colors do
      local seg = g.segments[i]
      local point = g.points[i]
      local next = g.points[i + 1]
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
  return g
end

local vec_add = Util.vec_add
local vec_scale = Util.vec_scale

function scene:one_line(color, vec1, vec2, existing)
  if not vec1 or not vec2 then
    return nil
  end
  if not existing then
    local l = Line.new(vec1, vec2, 2, colorfor(color))
    l:setThickness(3)
    return l
  else
    existing:setPoints(vec1, vec2)
    colorize(existing, color)
    return existing
  end
end

function scene:move()
  local bounce = false
  for i, v in ipairs(self.vecs) do
    if v:move(self.toward[i]) and (i == 1 or i == 4) then
      bounce = true
    end
  end
  -- not used during startup
  if bounce and self.next_color and not self.quiet then
    Sounds.play(ceil(self.next_color / set.color_multiplier))
  end
end

function scene:enterFrame(event)
  local last = table.remove(self.lines, 1)
  for i, l in ipairs(self.lines) do
    l.alpha = sqrt(i / set.history)
  end
  self.lines[#self.lines + 1] = self:line(nil, last)
  self.lines[#self.lines].alpha = 1
  self:move()
end

function scene:enterScene(event)
  self.lines = {}
  self.next_color = 1
  self.vecs = {}
  self.quiet = true
  for i = 1, self.points do
    self.vecs[i] = Vector.random(s, set)
  end
  self:move()
  for i = 1, set.history do
    local l = self:line(nil, nil)
    l.alpha = sqrt(i / set.history)
    self.lines[#self.lines + 1] = l
    self:move()
  end
  self.quiet = false
end

function scene:touch_magic(state, ...)
  self.toward = {}
  for i, v in pairs(state.points) do
    local lookup = { 1, 4, 2, 3 }
    if not v.done then
      self.toward[lookup[i] or 5] = v.current
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
