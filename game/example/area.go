package example

import "github.com/kybin/tiled"

var AxisArea = tiled.CreateArea([]tiled.Pos{{0, 1}, {1, 0}, {0, -1}, {-1, 0}})

var Axis2Area = tiled.CreateArea([]tiled.Pos{{0, 2}, {2, 0}, {0, -2}, {-2, 0}})

var DiagArea = tiled.CreateArea([]tiled.Pos{{1, 1}, {1, -1}, {-1, -1}, {-1, 1}})

var Diag2Area = tiled.CreateArea([]tiled.Pos{{2, 2}, {2, -2}, {-2, -2}, {-2, 2}})

var AroundArea = tiled.CreateArea([]tiled.Pos{{0, 1}, {1, 1}, {1, 0}, {1, -1}, {0, -1}, {-1, -1}, {-1, 0}, {-1, 1}})

var Around2Area = tiled.CreateArea([]tiled.Pos{
	{0, 2}, {1, 2}, {2, 2}, {2, 1}, {2, 0}, {2, -1}, {2, -2}, {1, -2},
	{0, -2}, {-1, -2}, {-2, -2}, {-2, -1}, {-2, 0}, {-2, 1}, {-2, 2}, {-1, 2},
})
