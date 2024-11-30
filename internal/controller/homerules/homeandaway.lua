function Evaluate(state, devices, _)
	if #devices == 0 then
	 	return state, 0, "no devices found"
	end
	local homeUsers = GetDevicesByState(devices, true)
	local wantHomeState = #homeUsers > 0
	local delay = 300
	if state.Home == wantHomeState then
        delay = 0
	end
    if wantHomeState then
        return { Home = true, Manual = true }, 0, "one or more users are home: " .. ListDevices(homeUsers)
    end
    return { Home = false, Manual = true }, delay, "all users are away: " .. ListDevices(devices)
end

function GetDevicesByState(list, state)
    local result = {}
    for _, obj in ipairs(list) do
        if obj.Home == state then
			table.insert(result, obj)
        end
    end
    return result
end

function ListDevices(devices)
	local result = ""
	for _, obj in ipairs(devices) do
		result = result .. obj.Name .. ", "
	end
	return result:sub(1, -3)
end