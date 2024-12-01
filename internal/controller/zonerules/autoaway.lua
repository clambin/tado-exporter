function Evaluate(_, state, devices, _)
        if #devices == 0 then
            return state, 0, "no devices found"
        end
        local homeUsers = FilterDevices(devices, true)
        if #homeUsers == 0 then
            return { Overlay = true, Heating = false }, 900, "all users are away"
        end
        if state.Heating then
            --- we didn't end the fire
            return state, 0, "one or more users are home: " .. ListDevices(homeUsers)
        end
        return { Overlay = false, Heating = true }, 0, "one or more users are home: " .. ListDevices(homeUsers)
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