import { CartesianGrid, Line, LineChart, YAxis } from "recharts"

import {
	ChartContainer,
	ChartLegend,
	ChartLegendContent,
	ChartTooltip,
	ChartTooltipContent,
	xAxis,
} from "@/components/ui/chart"
import {
	useYAxisWidth,
	cn,
	formatShortDate,
	toFixedFloat,
	chartMargin,
	decimalString,
} from "@/lib/utils"
import { ChartData } from "@/types"
import { memo, useMemo } from "react"
import { $genericSensorFilter } from "@/lib/stores"
import { useStore } from "@nanostores/react"

export default memo(function GenericSensorChart({ 
	chartData, 
	sensorName, 
	unit, 
	min, 
	max 
}: { 
	chartData: ChartData
	sensorName: string
	unit: string
	min: number
	max: number
}) {
	const filter = useStore($genericSensorFilter)
	const { yAxisWidth, updateYAxisWidth } = useYAxisWidth()

	if (chartData.systemStats.length === 0) {
		return null
	}

	/** Format generic sensor data for chart */
	const newChartData = useMemo(() => {
		const newChartData = { data: [], colors: {} } as {
			data: Record<string, number | string>[]
			colors: Record<string, string>
		}
		
		for (let data of chartData.systemStats) {
			let newData = { created: data.created } as Record<string, number | string>
			
			// Check if this sensor exists in the generic sensors data
			if (data.stats?.gs && data.stats.gs[sensorName]) {
				newData[sensorName] = data.stats.gs[sensorName].v
			}
			
			newChartData.data.push(newData)
		}

		// Set color for this sensor
		newChartData.colors[sensorName] = `hsl(${((sensorName.charCodeAt(0) * 137) % 360)}, 60%, 55%)`
		
		return newChartData
	}, [chartData, sensorName])

	// Format value for display
	const formatValue = (val: number) => {
		return toFixedFloat(val, 2) + " " + unit
	}

	// Calculate domain based on min/max with some padding
	const domain = useMemo(() => {
		const padding = (max - min) * 0.1
		return [Math.max(0, min - padding), max + padding]
	}, [min, max])

	return (
		<div>
			<ChartContainer
				className={cn("h-full w-full absolute aspect-auto bg-card opacity-0 transition-opacity", {
					"opacity-100": yAxisWidth,
				})}
			>
				<LineChart accessibilityLayer data={newChartData.data} margin={chartMargin}>
					<CartesianGrid vertical={false} />
					<YAxis
						direction="ltr"
						orientation={chartData.orientation}
						className="tracking-tighter"
						domain={domain}
						width={yAxisWidth}
						tickFormatter={(val) => updateYAxisWidth(formatValue(val))}
						tickLine={false}
						axisLine={false}
					/>
					{xAxis(chartData)}
					<ChartTooltip
						animationEasing="ease-out"
						animationDuration={150}
						content={
							<ChartTooltipContent
								labelFormatter={(_, data) => formatShortDate(data[0].payload.created)}
								contentFormatter={(item) => decimalString(item.value) + " " + unit}
								filter={filter}
							/>
						}
					/>
					<Line
						dataKey={sensorName}
						name={`${sensorName} (${unit})`}
						type="monotoneX"
						dot={false}
						strokeWidth={1.5}
						stroke={newChartData.colors[sensorName]}
						strokeOpacity={filter && !sensorName.toLowerCase().includes(filter.toLowerCase()) ? 0.1 : 1}
						activeDot={{ opacity: filter && !sensorName.toLowerCase().includes(filter.toLowerCase()) ? 0 : 1 }}
						isAnimationActive={false}
					/>
					<ChartLegend content={<ChartLegendContent />} />
				</LineChart>
			</ChartContainer>
		</div>
	)
})
