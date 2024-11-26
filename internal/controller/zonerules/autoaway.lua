function Evaluate(_, zone, devices)
        local allAway = true
        for _, device in ipairs(devices) do
        	if device.Home then
				allAway = false
        	end
        end
        if allAway == false then
                return zone, 0, "at least one user is home"
        end
        return "off", 3600, "all users are away"
end