function Evaluate(state, devices)
	if #devices == 0 then
	 	return state, 0, "no devices found"
	end
	local homeUsers = GetDevicesByState(devices, true)
	if #homeUsers == 0 then
		return "away", 300, "all users are away"
    end
	return "home", 0, "one or more users are home: " .. ListDevices(homeUsers)
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