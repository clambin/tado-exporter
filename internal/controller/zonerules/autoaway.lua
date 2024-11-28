function Evaluate(_, mode, devices, _)
        if #devices == 0 then
            return mode, 0, "no devices found"
        end
        if mode ~= "auto" and mode ~= "off" then
            return mode, 0, "no action required"
        end
        local homeUsers = FilterDevices(devices, true)
        if #homeUsers ~= 0 then
                return "auto", 0, "one or more users are home: " .. ListDevices(homeUsers)
        end
        return "off", 900, "all users are away"
end

function FilterDevices(devices, state)
    local filtered = {}
    for _, device in ipairs(devices) do
        if device.Home == state then
            table.insert(filtered, device)
        end
    end
    return filtered
end

function ListDevices(devices)
	local result = ""
	for _, obj in ipairs(devices) do
		result = result .. obj.Name .. ", "
	end
	return result:sub(1, -3)
end