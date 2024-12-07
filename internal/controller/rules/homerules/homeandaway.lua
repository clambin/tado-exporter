function Evaluate(state, devices, _)
	if #devices == 0 then
	 	return state, 0, "no devices found"
	end
	local homeUsers = FilterDevices(devices, true)
	local wantHomeState = #homeUsers > 0
	local reason = "all users are away: " .. ListDevices(devices)
	local delay = 300
	if wantHomeState then
	    reason = "one or more users are home: " .. ListDevices(homeUsers)
	    delay = 0
	end
	if state.Home == wantHomeState then
	    return state, 0, reason
	end
    return { Overlay = true, Home = wantHomeState }, delay, reason
end

function FilterDevices(devices, state)
    local result = {}
    for _, obj in ipairs(devices) do
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