set xlabel "Time (s)"
set title "Lap trace"
set autoscale x

set terminal png size 3000,400

set multiplot layout 2,1 rowsfirst
set yrange [0 : 80]
set ylabel "Speed (m/s)"
plot "trace.dat" using 1:2 title "Speed" axes x1y1 with lines

set yrange [0 : 1.1]
plot "trace.dat" using 1:3 title "Throttle" with lines linecolor rgb "green", \
	 "trace.dat" using 1:4 title "Brake" with lines linecolor rgb "red"

unset multiplot
