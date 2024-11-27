function Evaluate(mode, zone, devices)
        if #devices == 0 then
            return mode, 0, "no devices found"
        end
        local allAway = true
        for _, device in ipairs(devices) do
        	if device.Home then
				allAway = false
        	end
        end
        if allAway == false then
                return zone, 0, "at least one user is home"
        end
        return "off", 900, "all users are away"
end