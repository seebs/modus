local scene = {}

scene.meta = {
  name = "Bouncing Lines",
  description = "A line segment bounces around the screen leaving trails; multitouch adds or removes segments."
}

local s
local set
local rfuncs
local colorfor
local colorize

function scene:createScene(event)
  self.ids = self.ids or {}
  self.sorted_ids = self.sorted_ids or {}
  self.toward = self.toward or {}

  s = self.screen
  set = self.settings

  rfuncs = Rainbow.funcs_for(set.color_multiplier)
  colorfor = rfuncs.smooth
  colorize = rfuncs.smoothobj

  self.line_total = #Rainbow.hues * set.color_multiplier

  self.lines = {}
  -- so clicks have something to land on
  self.bg = display.newRect(s, 0, 0, s.size.x, s.size.y)
  self.bg:setFillColor(0, 0)
  self.view:insert(self.bg)
end

function scene:line(color, g)
  if not color then
    color = self.next_color or 1
    self.next_color = (color % self.line_total) + 1
  end
  if not g then
    g = display.newGroup(s)
  end
  g.segments = g.segments or {}
  for i = 1, #self.vecs - 1 do
    if g.segments[i] then
      self:one_line(color, self.vecs[i], self.vecs[i + 1], g.segments[i])
    else
      local l = self:one_line(color, self.vecs[i], self.vecs[i + 1])
      g.segments[i] = l
      g:insert(l)
    end
    g.segments[i]:redraw()
  end
  if #self.vecs > 2 then
    if g.segments[#self.vecs] then
      self:one_line(color, self.vecs[#self.vecs], self.vecs[1], g.segments[#self.vecs])
    else
      local l = self:one_line(color, self.vecs[#self.vecs], self.vecs[1])
      g.segments[#self.vecs] = l
      g:insert(l)
    end
    g.segments[#self.vecs]:redraw()
  end
  while #g.segments > #self.vecs or (#g.segments > 1 and #self.vecs == 2) do
    local l = table.remove(g.segments)
    l:removeSelf()
  end
  return g
end

function scene:one_line(color, vec1, vec2, existing)
  if not vec1 or not vec2 then
    return nil
  end
  if not existing then
    local l = Line.new(vec1, vec2, set.line_depth + 1, colorfor(color))
    l:setThickness(set.line_thickness)
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
    if v:move(self.toward[i]) then
      bounce = true
    end
  end
  -- not used during startup
  if bounce then
    Sounds.play(math.ceil(self.next_color / set.color_multiplier))
  end
end

function scene:enterFrame(event)
  local last = table.remove(self.lines, 1)
  table.insert(self.lines, self:line(nil, last))
  self:move()
end

function scene:enterScene(event)
  self.lines = {}
  self.next_color = 1
  self.vecs = {}
  s = Screen.new(self.view)
  for i = 1, 2 do
    self.vecs[i] = Vector.random(s, set)
  end
  for i = 1, self.line_total do
    local l = self:line(i, nil)
    self:move()
    table.insert(self.lines, l)
  end
  self.last_color = self.line_total
end

function scene:touch_magic(state, ...)
  self.toward = {}
  local highest = 0
  for i, v in pairs(state.points) do
    if not v.done then
      self.toward[i] = v.current
      if i > highest then
        highest = i
      end
    end
  end
  while highest > #self.vecs do
    local v = Vector.random(s, set)
    local t = self.toward[#self.vecs + 1]
    -- start it at the new finger
    if t then
      v.x = t.x
      v.y = t.y
    end
    table.insert(self.vecs, v)
  end
  if highest == 0 and state.peak < #self.vecs and #self.vecs > 2 then
    table.remove(self.vecs, 1)
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
