set title "Lap trace"
set xlabel "Lap distance (m)"
set autoscale x

set terminal png size 3000,400

set multiplot layout 2,1 rowsfirst
set yrange [0 : 80]
plot "trace.dat" using 1:2 title "Speed" axes x1y1 with lines
unset title
set ylabel "Speed (m/s)"

set style fill transparent solid 0.5 noborder
set yrange [0 : 1.1]
plot "trace.dat" using 1:3 title "Throttle" with filledcurves below x1 linecolor rgb "forest-green", \
     "trace.dat" using 1:4 title "Brake" with filledcurves below x1 linecolor rgb "red"

unset multiplot
