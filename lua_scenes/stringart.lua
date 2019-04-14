local scene = {}

scene.meta = {
  name = "String Art",
  description = "Lines wander around the screen, creating patterns in their trails."
}

local s
local set
local rfuncs
local colorfor
local colorize
local random = math.random
local ceil = math.ceil
local floor = math.floor
local table_remove = table.remove
local dist = Util.dist

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
end

function scene:line(point, color, l)
  if not color then
    color = point.next_color or 1
    point.next_color = (color % self.line_total) + 1
  end
  local vec1 = point.vecs[1]
  local vec2 = point.vecs[2]
  if not l then
    l = Line.new(vec1, vec2, set.line_depth, colorfor(color))
    s:insert(l)
    l:setThickness(set.line_thickness - 0.5)
    l:redraw()
  else
    l:setPoints(vec1, vec2)
    colorize(l, color)
    l:redraw()
  end
  return l
end

-- either is_boring(array of pairs, nil, v3) or is_boring(v1, v2, v3)
local function is_boring(v1, v2, v3)
  if not v2 then
    for i = 1, #v1 do
      -- Util.printf("  %d/%d: %s, %s",
        -- i, #v1, tostring(v1[i][1]), tostring(v1[i][2]))
      if is_boring(v1[i][1], v1[i][2], v3) then
        return true
      end
    end
    return false
  end
  if v3.x == v1.x and v3.x == v2.x then
    return true
  end
  if v3.y == v1.y and v3.y == v2.y then
    return true
  end
  if v3.x == v1.x and v3.y == v1.y then
    return true
  end
  if v3.x == v2.x and v3.y == v2.y then
    return true
  end
  -- Util.printf("not boring: %s, %s, %s", tostring(v1), tostring(v2), tostring(v3))
  return false
end

local function is_corner(vec)
  return (vec.x == 0 or vec.x == s.size.x) and (vec.y == 0 or vec.y == s.size.y)
end

function scene:find_interest(pairs, out, index)
  index = ((index - 1) % #self.interesting_points) + 1
  local ip = self.interesting_points[index]
  local r
  local magnet = self.toward[index] or self.toward[1]
  -- pick the closest point, if it works
  if magnet then
    local best
    for i = 1, #ip do
      local d2 = dist(ip[i], magnet)
      if not best or d2 < best then
        r = i
	best = d2
      end
    end
  end
  if not r then
    if out.index then
      r = (out.index % #ip) + 1
    else
      r = random(#ip)
    end
  end
  local r_orig = r
  local v3 = ip[r]
  -- we got an array of pairs, probably
  while is_boring(pairs, nil, v3) do
    r = (r % #ip) + 1
    -- we give up
    if r == r_orig then
      -- Util.printf("can't find interesting point for:")
      -- for i = 1, #pairs do
	-- Util.printf("  %s + %s", tostring(pairs[i][1]), tostring(pairs[i][2]))
      -- end
      break
    end
    v3 = ip[r]
  end
  out.index = r
  out.x = v3.x
  out.y = v3.y
end

--
-- normally:
-- point 1 moves towards point 3; point 2 is fixed.
-- after point 1 reaches point 3, that location becomes point 2, point 2's
-- location becomes point 1, and a new point 3 is generated.
--
-- if move_both:
-- point 1 moves towards point 3; point 2 moves towards point 1's starting
-- location. when point 1 arrives, point 2 is assumed to be Very Close to
-- where point 1 started. So we don't shuffle; we just make a new point 3,
-- and continue
--
function scene:move(point)
  local bounce = false
  local next = self.points[(point.index % #self.points) + 1]
  if point.move_both then
    point.vecs[2]:move()
  end
  if point.vecs[1]:move() then
    local out
    if point.move_both then
      -- handle roundoff error: move the second point to where it thought it
      -- was going
      point.vecs[2].x = point.orig[1].x
      point.vecs[2].y = point.orig[1].y
      point.vecs[1].x = point.orig[3].x
      point.vecs[1].y = point.orig[3].y
      -- 50-50 chance of which end we consider the new "moving" end
      if random(2) == 2 then
        local t = point.vecs[1]
	point.vecs[1] = point.vecs[2]
	point.vecs[2] = t
      end
      -- and mark the new locations as the origin
      point.orig[1].x = point.vecs[1].x
      point.orig[1].y = point.vecs[1].y
      point.orig[2].x = point.vecs[2].x
      point.orig[2].y = point.vecs[2].y
    else
      -- rotate 1 <- 2 <- 3 <- 1
      out = table.remove(point.orig, 1)
      point.orig[3] = out
      out = table_remove(point.vecs, 1)
      point.vecs[3] = out
      point.vecs[3].index = point.vecs[2].index
    end

    clash = true
    counter = 0
    local clashes = {
	point.vecs,
      }
    local repeated = { x = point.vecs[3].x, y = point.vecs[3].y }
    local last_move_both = point.move_both
    if random(3) == 1 and not self.toward[point.index] then
      -- never do this twice in a row
      if point.corners then
        self:find_interest(clashes, point.vecs[3], point.index)
        point.corners = false
      else
        self:find_interest(clashes, point.vecs[3], 1)
        point.corners = true
      end
    else
      self:find_interest(clashes, point.vecs[3], point.index)
    end
    point.vecs[1]:set_target(point.vecs[3], point.steps)
    point.orig[3].x = point.vecs[3].x
    point.orig[3].y = point.vecs[3].y
    point.move_both = true
    -- if we picked the same three points somehow, change the behavior
    if (point.vecs[3].x == repeated.x and point.vecs[3].y == repeated.y) or
       (point.vecs[2].x == repeated.x and point.vecs[2].y == repeated.y) then
      point.move_both = not last_move_both
    elseif is_corner(point.vecs[1]) and is_corner(point.vecs[3]) then
      -- we only want to draw straight instead of curved if:
      -- the new point is a corner
      -- the point we'd be drawing from is a corner on one of the same two edges
      if point.vecs[1].x == point.vecs[3].x or point.vecs[1].y == point.vecs[3].y then
	if random(2) == 2 then
          point.move_both = false
	end
      end
    end
    if point.move_both then
      point.vecs[2]:set_target(point.vecs[1], point.steps)
    end
    if point.next_color and not self.quiet then
      Sounds.playoctave(math.ceil(point.next_color / set.color_multiplier), point.index - 1)
    end
    point.next_color = (point.next_color % self.line_total) + 1
    if point.index > 2 and random(6) == 1 then
      point.index = point.index + 1
    end
  end
end

function scene:advance(point)
  local last
  -- take the oldest line
  if #point.lines >= self.lines_per or point.done then
    last = table_remove(point.lines, 1)
  end
  if point.done then
    if last then
      last:removeSelf()
    end
  else
    point.lines[#point.lines + 1] = self:line(point, nil, last)
    self:move(point)
  end
end

function scene:enterFrame(event)
  local removes = {}
  for i = 1, #self.points do
    local point = self.points[i]
    self:advance(point)
    if #point.lines == 0 then
      removes[#removes + 1] = i
    end
  end
  for i = #removes, 1, -1 do
    self.points[removes[i]] = nil
  end
end

function scene:addpoint(i, p1, p2)
  local offset = ceil(i * self.line_total / set.points / 2)
  local point = {
    vecs = {},
    orig = {},
    lines = {},
    next_color = 1,
    index = i,
    prev = p2,
    pprev = p1,
    steps = self.lines_per + i * 2,
  } 
  local ip = self.interesting_points[((i - 1) % #self.interesting_points) + 1]
  local r = random(#ip)
  point.vecs[1] = Vector.coords(s, set, ip[r])
  r = (r % #ip) + 1
  point.orig[1] = point.vecs[1]:copy()
  point.vecs[2] = Vector.coords(s, set, ip[r])
  r = (r % #ip) + 1
  point.orig[2] = point.vecs[2]:copy()
  point.vecs[3] = Vector.coords(s, set, ip[r])
  self:find_interest({ point.vecs }, point.vecs[3], i)
  point.orig[3] = point.vecs[3]:copy()
  point.vecs[1]:set_target(point.vecs[3], point.steps)
  self.points[i] = point
  -- Util.printf("points[%d] = { %s, %s, %s }",
    -- i,
    -- tostring(point.vecs[1]),
    -- tostring(point.vecs[2]),
    -- tostring(point.vecs[3]))
end

function scene:enterScene(event)
  self.lines = {}
  self.points = {}
  self.next_point = 1
  s = Screen.new(self.view)
  self.lines_per = floor(self.line_total * 3 / 4)
  self.interesting_points = {
    {
      Vector.coords(s, set, 0,              0),
      Vector.coords(s, set, s.size.x,       0),
      Vector.coords(s, set, s.size.x,       s.size.y),
      Vector.coords(s, set, 0,              s.size.y),
    },
    {
      Vector.coords(s, set, s.center.x,     0),
      Vector.coords(s, set, 0,              s.center.y),
      Vector.coords(s, set, s.center.x,     s.size.y),
      Vector.coords(s, set, s.size.x,       s.center.y),
    },
    {
      Vector.coords(s, set, s.size.x / 3,     0),
      Vector.coords(s, set, s.size.x,         s.size.y / 3),
      Vector.coords(s, set, s.size.x * 2 / 3,     s.size.y),
      Vector.coords(s, set, 0,                s.size.y * 2 / 3),
    },
    {
      Vector.coords(s, set, 0,              0),
      Vector.coords(s, set, s.size.x,       0),
      Vector.coords(s, set, 0,              s.size.y),
      Vector.coords(s, set, s.size.x,       s.size.y),
      Vector.coords(s, set, s.center.x,     0),
      Vector.coords(s, set, 0,              s.center.y),
      Vector.coords(s, set, s.center.x,     s.size.y),
      Vector.coords(s, set, s.size.x,       s.center.y),
    },
    {
      Vector.coords(s, set, s.size.x * 3 / 4,     0),
      Vector.coords(s, set, s.size.x,         s.size.y * 3 / 4),
      Vector.coords(s, set, s.size.x / 4,     s.size.y),
      Vector.coords(s, set, 0,                s.size.y / 4),
    },
    {
      Vector.coords(s, set, s.size.x / 4,     0),
      Vector.coords(s, set, s.size.x * 3 / 4, 0),
      Vector.coords(s, set, s.size.x,         s.size.y / 4),
      Vector.coords(s, set, s.size.x,         s.size.y * 3 / 4),
      Vector.coords(s, set, s.size.x / 4,     s.size.y),
      Vector.coords(s, set, s.size.x * 3 / 4, s.size.y),
      Vector.coords(s, set, 0,                s.size.y / 4),
      Vector.coords(s, set, 0,                s.size.y * 3/ 4),
    },
  }
  self.quiet = true
  for i = 1, 2 do
    self:addpoint(i)
  end
  for i = 3, set.points do
    self:addpoint(i, self.points[i - 2], self.points[i - 1])
  end
  local last = self.points[#self.points]
  self.points[1].prev = last
  self.points[2].pprev = last
  last = self.points[#self.points - 1]
  self.points[1].pprev = last
  self.points[2].prev = self.points[1]

  -- advance each line by offset plus the lines_per default
  for i = 1, #self.points do
    local offset = ceil(i * self.line_total / set.points)
    local point = self.points[i]
    -- now rotate them ahead a bit
    for j = 1, self.lines_per + offset do
      self:advance(point)
    end
  end
  -- there are three points under consideration:
  -- point 1 moves towards point 3
  -- point 2 is the other end of the line
  -- when point 1 arrives at point 3, point 2 becomes point 1, point 3
  -- becomes point 2, and a new point 3 is generated
  self.quiet = false
end

function scene:touch_magic(state, ...)
  self.toward = self.toward or {}
  local highest = 0
  for i = 1, state.peak do
    local v = state.points[i]
    if v and not v.done then
      self.toward[i] = v.current
      if i > highest then
        highest = i
      end
    else
      self.toward[i] = nil
    end
  end
  while highest > #self.points do
    self:addpoint(#self.points + 1, self.points[#self.points], self.points[#self.points - 1])
    -- touch up the back-references
    self.points[1].pprev = self.points[#self.points - 2]
    self.points[2].pprev = self.points[#self.points - 1]
    self.points[1].prev = self.points[#self.points - 1]
  end
  -- fade away old points
  if highest == 0 and state.peak < #self.points and #self.points > set.points then
    local point = self.points[#self.points]
    point.done = true
  end
end

function scene:exitScene(event)
  self.sorted_ids = {}
  self.toward = {}
  self.view.alpha = 0
  for i = 1, #self.points do
    local point = self.points[i]
    for j = 1, #point.lines do
      local l = point.lines[j]
      l:removeSelf();
    end
    point.lines = {}
  end
  self.points = {}
end

function scene:destroyScene(event)
  self.points = nil
end

return scene
