function Evaluate(_, state, devices, _)
        if #devices == 0 then
            return state, 0, "no devices found"
        end
        local homeUsers = FilterDevices(devices, true)
        local wantHeating = #homeUsers > 0
        if #homeUsers > 0 then
            reason = "one or more users are home: " .. ListDevices(homeUsers)
        else
            reason = "all users are away"
        end
        if state.Heating == wantHeating then
            return state, 0, reason
        end

        if #homeUsers == 0 then
            return { Overlay = true, Heating = false }, 900, reason
        end
        return { Overlay = false, Heating = true }, 0, reason
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