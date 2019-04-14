local scene = {}

scene.meta = {
  name = "Raindrops",
  description = "Raindrops splash and fade, following touch events."
}

local dist = Util.dist
local random = math.random
local floor = math.floor
local ceil = math.ceil

local s
local set

scene.future_drops = {}

function scene.setDropVisible(drop, hidden)
  drop.innerc.isVisible = hidden
  drop.innerh.isVisible = hidden
  drop.outerc.isVisible = hidden
  -- drop.outerh.isVisible = hidden
end

function scene.setDropScale(drop, reset_or_main, inner, outer)
  if reset_or_main == true then
    drop.scale = 1
    drop.inner_scale = 1
    drop.outer_scale = 1
    drop.innerc.xScale = drop.iscale
    drop.innerc.yScale = drop.iscale
    drop.innerh.xScale = drop.iscale
    drop.innerh.yScale = drop.iscale
    drop.outerc.xScale = drop.iscale
    drop.outerc.yScale = drop.iscale
    -- drop.outerh.xScale = drop.iscale
    -- drop.outerh.yScale = drop.iscale
  else
    local is = drop.iscale * inner * reset_or_main
    local os = drop.iscale * outer * reset_or_main
    drop.scale = reset_or_main
    drop.inner_scale = inner
    drop.outer_scale = outer
    drop.innerc.xScale = is
    drop.innerc.yScale = is
    drop.innerh.xScale = is
    drop.innerh.yScale = is
    drop.outerc.xScale = os
    drop.outerc.yScale = os
    -- drop.outerh.xScale = os
    -- drop.outerh.yScale = os
  end
end

function scene.setDropXY(drop, x, y)
  drop.x = x
  drop.y = y
  drop.innerc.x, drop.innerc.y = x, y
  drop.innerh.x, drop.innerh.y = x, y
  drop.outerc.x, drop.outerc.y = x, y
  -- drop.outerh.x, drop.outerh.y = x, y
end

function scene.setDropAlpha(drop, reset_or_main, inner, outer)
  if reset_or_main == true then
    drop.innerc.alpha = 1
    drop.innerh.alpha = .8
    drop.outerc.alpha = 1
    -- drop.outerh.alpha = .64
  else
    drop.innerc.alpha = reset_or_main * inner
    drop.innerh.alpha = reset_or_main * inner * .8
    drop.outerc.alpha = reset_or_main * outer
    -- drop.outerh.alpha = reset_or_main * outer * .64
  end
end

function scene:createScene(event)
  self.drops = {}
  s = self.screen
  set = self.settings
  -- so clicks have something to land on
  self.bg = display.newRect(s, 0, 0, s.size.x, s.size.y)
  self.bg:setFillColor(0, 0)
  self.last_color = 1
  s:insert(self.bg)
  self.spare_drops = {}
  self.last_hue = nil
  self.sheetc = graphics.newImageSheet("drop_widec.png", { width = 512, height = 512, numFrames = 1 })
  self.sheeth = graphics.newImageSheet("drop_wideh.png", { width = 512, height = 512, numFrames = 1 })
  self.iscale = 200 / 512
  self.oscale = 300 / 512
  for i = 1, set.total_drops do
    local d = {
      iscale = self.iscale,
      oscale = self.oscale,
      setVisible = self.setDropVisible,
      setScale = self.setDropScale,
      setAlpha = self.setDropAlpha,
      setXY = self.setDropXY,
    }
    local img
    d.hue = ((i - 1) % #Rainbow.hues) + 1
    d.id = i
    d.octave = floor((i - 1) / #Rainbow.hues)
    local r, g, b = unpack(Rainbow.color(i))

    img = display.newImage(self.sheetc, 1)
    img:setFillColor(r, g, b)
    img.blendMode = 'add'
    s:insert(img)
    d.innerc = img

    img = display.newImage(self.sheeth, 1)
    img.blendMode = 'add'
    s:insert(img)
    d.innerh = img

    img = display.newImage(self.sheeth, 1)
    img:setFillColor(r, g, b)
    img.blendMode = 'add'
    s:insert(img)
    d.outerc = img

    -- img = display.newImage(self.sheeth, 1)
    -- img.blendMode = 'add'
    -- img:setFillColor(255, 180)
    -- s:insert(img)
    -- d.outerh = img

    d:setScale(true)
    d:setAlpha(true)
    d:setVisible(false)

    table.insert(self.spare_drops, d)
  end
end

function scene:do_drops()
  local spares = {}
  for i, d in ipairs(self.drops) do
    d:setScale(d.scale + 0.01, d.inner_scale + 0.008, d.outer_scale + 0.02)
    d.growth = d.growth + 1
    local halfway = d.max_growth / 2
    if d.growth >= d.max_growth then
      d:setVisible(false)
      table.insert(spares, i)
    elseif d.growth >= halfway then
      local mod = 1 - ((d.growth - halfway) / halfway)
      local sqmod = math.sqrt(mod)
      d:setAlpha(mod, sqmod, sqmod)
    else
      d:setAlpha(true)
    end
  end
  while #spares > 0 do
    local idx = table.remove(spares)
    table.insert(self.spare_drops, table.remove(self.drops, idx))
  end
  self.drop_cooldown = self.drop_cooldown - 1
  if #self.spare_drops > 0 and random(#self.spare_drops) > set.drop_threshold and self.drop_cooldown < 1 then
    local d = table.remove(self.spare_drops, 1)
    if #self.spare_drops > 1 then
      local counter = #self.spare_drops
      while counter > 0 and d.hue == self.last_hue do
        table.insert(self.spare_drops, d)
	d = table.remove(self.spare_drops, 1)
	counter = counter - 1
      end
    end
    self.last_hue = d.hue
    Sounds.playoctave(d.hue, d.octave)
    local new_point
    if #self.future_drops > 0 then
      new_point = table.remove(self.future_drops, 1)
    else
      new_point = { x = random((s.size.x - 150) + 75),
                    y = random((s.size.y - 200) + 100) }
    end
    self.drop_cooldown = random(set.max_cooldown - set.min_cooldown) + set.min_cooldown
    -- faster when there's pending action...
    if #self.future_drops > 0 then
      local scale = 0.5
      local weighted_min = set.min_cooldown * scale
      self.drop_cooldown = ceil((self.drop_cooldown + weighted_min) / (scale + 1))
    end
    d:setXY(new_point.x, new_point.y)
    local range = set.max_growth - set.min_growth
    local scale = random(range)
    d.max_growth = scale + set.min_growth
    d.factor = (scale / range) * 0.2
    d:setVisible(true)
    d:setAlpha(true)
    d:setScale(0.05 + d.factor, .3, 1)
    d.growth = 0
    table.insert(self.drops, d)
  end
end

function scene:enterFrame(event)
  self:do_drops()
end

local last_drops = {}

function scene:touch_magic(state, ...)
  if state.events > 0 then
    for i, e in pairs(state.points) do
      if e.events > 0 then
        if not e.done then
	  local last = last_drops[i]
	  local next = { x = e.current.x, y = e.current.y, stamp = e.stamp }
	  if not last or dist(last, e.current) > 80 or e.stamp - last.stamp > 100 then
	    table.insert(self.future_drops, next)
	    last_drops[i] = next
	  end
        else
	  last_drops[i] = nil
	end
      end
    end
  end
end

function scene:willEnterScene(event)
end

function scene:enterScene(event)
  last_drops = {}
  self.future_drops = {}
  self.drop_cooldown = 0
end

function scene:didExitScene(event)
  local move_these = {}
  for i, d in ipairs(self.drops) do
    d:setVisible(false)
    d:setScale(true)
    table.insert(move_these, i)
  end
  while #move_these > 0 do
    table.insert(self.spare_drops, table.remove(self.drops, table.remove(move_these)))
  end
end

function scene:destroyScene(event)
  self.drops = nil
  self.spare_drops = nil
  self.bg = nil
  self.sheetc = nil
  self.sheeth = nil
end

return scene
