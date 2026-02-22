import { useState, useEffect } from 'react';
import { 
  Database, 
  Settings, 
  FileText, 
  Key, 
  Share2, 
  Network,
  Layers
} from 'lucide-react';

const App = () => {
  const [isLoading, setIsLoading] = useState(true);
  const [activeSegment, setActiveSegment] = useState(0);

  // Loading screen: dismiss after delay
  useEffect(() => {
    const t = setTimeout(() => setIsLoading(false), 2200);
    return () => clearTimeout(t);
  }, []);

  // Cycle the status ticker
  useEffect(() => {
    const interval = setInterval(() => {
      setActiveSegment((prev) => (prev + 1) % 4);
    }, 3000);
    return () => clearInterval(interval);
  }, []);

  const dbTypes = [
    { name: 'SQL', icon: Database, color: 'text-indigo-600', description: 'Relational logic' },
    { name: 'NoSQL', icon: FileText, color: 'text-violet-600', description: 'Document stores' },
    { name: 'Caching', icon: Key, color: 'text-orange-500', description: 'Key-Value pairs' },
    { name: 'Graph', icon: Network, color: 'text-blue-500', description: 'Relationship nodes' },
  ];

  // The lemniscate (figure-eight) path
  const infinityPath = "M 400,200 C 550,50 750,50 750,200 C 750,350 550,350 400,200 C 250,50 50,50 50,200 C 50,350 250,350 400,200 Z";

  // Loading screen: infinity loop + OmniQL badge
  if (isLoading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-indigo-50/50 flex flex-col items-center justify-center p-8 font-sans">
        <div className="absolute inset-0 opacity-[0.04]" style={{ backgroundImage: 'linear-gradient(rgba(0,0,0,.2) 1px, transparent 1px), linear-gradient(90deg, rgba(0,0,0,.2) 1px, transparent 1px)', backgroundSize: '24px 24px' }} />
        <div className="relative z-10 w-full max-w-[750px] h-[380px] flex items-center justify-center">
          <svg
            viewBox="0 0 800 400"
            className="w-full h-full drop-shadow-2xl"
            xmlns="http://www.w3.org/2000/svg"
          >
            <defs>
              <linearGradient id="loadInfinityGrad" x1="0%" y1="0%" x2="100%" y2="0%">
                <stop offset="0%" stopColor="#312e81" />
                <stop offset="50%" stopColor="#7c3aed" />
                <stop offset="100%" stopColor="#f97316" />
              </linearGradient>
              <linearGradient id="loadInfinityGradAnim" x1="0%" y1="0%" x2="100%" y2="0%">
                <stop offset="0%" stopColor="#4f46e5">
                  <animate attributeName="stop-color" values="#312e81;#6366f1;#312e81" dur="4s" repeatCount="indefinite" />
                </stop>
                <stop offset="50%" stopColor="#a78bfa">
                  <animate attributeName="stop-color" values="#7c3aed;#c4b5fd;#7c3aed" dur="4s" repeatCount="indefinite" />
                </stop>
                <stop offset="100%" stopColor="#fb923c">
                  <animate attributeName="stop-color" values="#f97316;#fdba74;#f97316" dur="4s" repeatCount="indefinite" />
                </stop>
              </linearGradient>
              <filter id="loadGlow">
                <feGaussianBlur stdDeviation="3" result="coloredBlur" />
                <feMerge>
                  <feMergeNode in="coloredBlur" />
                  <feMergeNode in="SourceGraphic" />
                </feMerge>
              </filter>
              <filter id="loadSoftShadow" x="-20%" y="-20%" width="140%" height="140%">
                <feDropShadow dx="0" dy="2" stdDeviation="8" floodOpacity="0.12" />
              </filter>
            </defs>
            {/* Infinity loop drawn progressively (being "made") */}
            <path
              d={infinityPath}
              fill="none"
              stroke="url(#loadInfinityGrad)"
              strokeWidth="58"
              strokeLinecap="round"
              strokeLinejoin="round"
              pathLength="1"
              strokeDasharray="1"
              className="load-path-draw"
              opacity="0.25"
              filter="url(#loadSoftShadow)"
            />
            <path
              d={infinityPath}
              fill="none"
              stroke="url(#loadInfinityGradAnim)"
              strokeWidth="50"
              strokeLinecap="round"
              strokeLinejoin="round"
              pathLength="1"
              strokeDasharray="1"
              className="load-path-draw"
              opacity="0.95"
              filter="url(#loadSoftShadow)"
            />
            <path
              d={infinityPath}
              fill="none"
              stroke="rgba(255,255,255,0.35)"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              pathLength="1"
              strokeDasharray="1"
              className="load-path-draw"
            />
          </svg>
        </div>
        <div className="relative z-10 mt-6 flex flex-col items-center gap-6">
          <div className="bg-white/95 backdrop-blur-sm px-10 py-4 rounded-full shadow-xl border border-slate-200/80 ring-2 ring-indigo-500/10 flex items-center space-x-3 animate-[fadeIn_0.5s_ease-out]">
            <div className="relative">
              <Database className="text-indigo-600" size={28} />
              <div className="absolute -top-0.5 -right-0.5 w-3 h-3 bg-emerald-500 rounded-full ring-2 ring-white animate-pulse" />
            </div>
            <span className="font-extrabold text-slate-800 tracking-tight text-xl uppercase bg-gradient-to-r from-indigo-600 to-orange-500 bg-clip-text text-transparent">OmniQL</span>
          </div>
          <div className="h-1 w-32 rounded-full bg-slate-200 overflow-hidden">
            <div className="h-full w-1/3 rounded-full bg-gradient-to-r from-indigo-500 to-orange-500 animate-[loadingBar_2s_ease-in-out_forwards]" />
          </div>
        </div>
        <style dangerouslySetInnerHTML={{ __html: `
          @keyframes fadeIn { from { opacity: 0; transform: scale(0.96); } to { opacity: 1; transform: scale(1); } }
          @keyframes loadingBar { 0% { width: 0%; } 70% { width: 90%; } 100% { width: 100%; } }
          @keyframes drawPath { from { stroke-dashoffset: 1; } to { stroke-dashoffset: 0; } }
          .load-path-draw { animation: drawPath 2s ease-out forwards; }
        `}} />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col items-center justify-center p-8 font-sans">
      <div className="max-w-4xl w-full text-center space-y-8">
        {/* Header Section */}
        <div className="space-y-4">
          <h1 className="text-5xl font-extrabold tracking-tight text-slate-900 bg-clip-text text-transparent bg-gradient-to-r from-indigo-600 to-orange-500">
            OmniQL
          </h1>
          <p className="text-xl text-slate-600 max-w-2xl mx-auto leading-relaxed">
            A universal database adapter and query engine. One interface, any database.
          </p>
        </div>

        {/* The Animated Infinity Loop Canvas */}
        <div className="relative h-[450px] w-full flex items-center justify-center overflow-hidden rounded-3xl bg-gradient-to-br from-slate-50 via-white to-indigo-50/40 shadow-2xl border border-slate-200/80 ring-1 ring-white/50">
          {/* Subtle grid overlay */}
          <div className="absolute inset-0 opacity-[0.03]" style={{ backgroundImage: 'linear-gradient(rgba(0,0,0,.15) 1px, transparent 1px), linear-gradient(90deg, rgba(0,0,0,.15) 1px, transparent 1px)', backgroundSize: '24px 24px' }} />
          <svg
            viewBox="0 0 800 400"
            className="w-full h-full max-w-[750px] drop-shadow-2xl relative z-10"
            xmlns="http://www.w3.org/2000/svg"
          >
            <defs>
              <linearGradient id="infinityGradient" x1="0%" y1="0%" x2="100%" y2="0%">
                <stop offset="0%" stopColor="#312e81" />
                <stop offset="50%" stopColor="#7c3aed" />
                <stop offset="100%" stopColor="#f97316" />
              </linearGradient>
              <linearGradient id="infinityGradientAnimated" x1="0%" y1="0%" x2="100%" y2="0%">
                <stop offset="0%" stopColor="#4f46e5">
                  <animate attributeName="stop-color" values="#312e81;#6366f1;#312e81" dur="4s" repeatCount="indefinite" />
                </stop>
                <stop offset="50%" stopColor="#a78bfa">
                  <animate attributeName="stop-color" values="#7c3aed;#c4b5fd;#7c3aed" dur="4s" repeatCount="indefinite" />
                </stop>
                <stop offset="100%" stopColor="#fb923c">
                  <animate attributeName="stop-color" values="#f97316;#fdba74;#f97316" dur="4s" repeatCount="indefinite" />
                </stop>
              </linearGradient>

              <filter id="glow">
                <feGaussianBlur stdDeviation="3" result="coloredBlur"/>
                <feMerge>
                  <feMergeNode in="coloredBlur"/>
                  <feMergeNode in="SourceGraphic"/>
                </feMerge>
              </filter>
              <filter id="softShadow" x="-20%" y="-20%" width="140%" height="140%">
                <feDropShadow dx="0" dy="2" stdDeviation="8" floodOpacity="0.12"/>
              </filter>
              <filter id="hubShadow" x="-50%" y="-50%" width="200%" height="200%">
                <feDropShadow dx="0" dy="4" stdDeviation="12" floodColor="#1e1b4b" floodOpacity="0.15"/>
              </filter>
            </defs>

            {/* Outer glow / shadow layer */}
            <path
              d={infinityPath}
              fill="none"
              stroke="url(#infinityGradient)"
              strokeWidth="58"
              strokeLinecap="round"
              opacity="0.25"
              filter="url(#softShadow)"
            />
            {/* Main infinity stroke */}
            <path
              d={infinityPath}
              fill="none"
              stroke="url(#infinityGradientAnimated)"
              strokeWidth="50"
              strokeLinecap="round"
              className="opacity-95"
              filter="url(#softShadow)"
            />
            {/* Inner highlight */}
            <path
              d={infinityPath}
              fill="none"
              stroke="rgba(255,255,255,0.35)"
              strokeWidth="2"
              strokeLinecap="round"
              className="pointer-events-none"
            />

            {/* Moving Database Icons */}
            {dbTypes.map((db, i) => (
              <g 
                key={i} 
                className="path-animation" 
                style={{ 
                  offsetPath: `path('${infinityPath}')`,
                  animationDelay: `-${i * 2.5}s`, 
                  animationDuration: '10s'
                }}
              >
                <foreignObject 
                  width="48" 
                  height="48" 
                  x="-24" 
                  y="-24"
                >
                  <div className="flex items-center justify-center w-full h-full bg-white rounded-full shadow-lg border-2 border-slate-100 ring-2 ring-white/80 transform scale-95 hover:scale-100 transition-transform duration-200">
                    <db.icon size={24} className={db.color} />
                  </div>
                </foreignObject>
              </g>
            ))}

            {/* Light trails */}
            {[0, 1, 2, 3].map((i) => (
              <g 
                key={`trail-${i}`}
                className="path-animation"
                style={{ 
                  offsetPath: `path('${infinityPath}')`,
                  animationDelay: `-${i * 2.3}s`,
                  animationDuration: '7s'
                }}
              >
                <circle r="5" fill="white" filter="url(#glow)" opacity="0.9" />
              </g>
            ))}

            {/* Left Hub: Adapter Engine */}
            <g transform="translate(180, 200)" filter="url(#hubShadow)">
              <circle r="68" fill="white" stroke="rgba(99,102,241,0.2)" strokeWidth="2" />
              <circle r="65" fill="url(#infinityGradient)" fillOpacity="0.08" />
              <g className="animate-[spin_15s_linear_infinite]">
                <Settings size={80} strokeWidth={1.2} className="text-indigo-700" style={{ transform: 'translate(-40px, -40px)' }} />
              </g>
            </g>

            {/* Right Hub: Query Engine */}
            <g transform="translate(620, 200)" filter="url(#hubShadow)">
              <circle r="68" fill="white" stroke="rgba(249,115,22,0.2)" strokeWidth="2" />
              <circle r="65" fill="url(#infinityGradient)" fillOpacity="0.08" />
              <Layers 
                size={80} 
                strokeWidth={1.2} 
                className="text-orange-500" 
                style={{ transform: 'translate(-40px, -40px)' }} 
              />
            </g>
          </svg>
        </div>

        {/* Features Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8 mt-12 text-left">
          <div className="p-8 bg-white rounded-3xl shadow-sm border border-slate-100 transition-all hover:shadow-xl">
            <div className="w-12 h-12 bg-indigo-50 rounded-2xl flex items-center justify-center mb-4">
              <Share2 className="text-indigo-600" size={24} />
            </div>
            <h3 className="text-xl font-bold text-slate-900 mb-3">Dialect Abstraction</h3>
            <p className="text-slate-600 leading-relaxed">
              Eliminate database friction. OmniQL translates your abstract queries into high-performance native code for any relational or NoSQL target.
            </p>
          </div>
          <div className="p-8 bg-white rounded-3xl shadow-sm border border-slate-100 transition-all hover:shadow-xl">
            <div className="w-12 h-12 bg-orange-50 rounded-2xl flex items-center justify-center mb-4">
              <Layers className="text-orange-600" size={24} />
            </div>
            <h3 className="text-xl font-bold text-slate-900 mb-3">Infinite Portability</h3>
            <p className="text-slate-600 leading-relaxed">
              Future-proof your infrastructure. Move seamlessly between SQL, Graph, or Document stores without modifying a single line of application code.
            </p>
          </div>
        </div>

        {/* Dynamic Status Ticker */}
        <div className="bg-slate-950 text-indigo-100 py-4 px-8 rounded-2xl flex items-center justify-between max-w-2xl mx-auto border border-white/10 shadow-inner">
          <div className="flex items-center space-x-3 shrink-0">
            <div className="w-2.5 h-2.5 rounded-full bg-green-400 animate-pulse"></div>
            <span className="text-xs font-bold uppercase tracking-widest text-indigo-400">Engine Ticker:</span>
          </div>
          <div className="flex-1 ml-6 text-sm font-mono overflow-hidden whitespace-nowrap">
            <span className="inline-block animate-[marquee_25s_linear_infinite] text-indigo-200">
              Adapting to {dbTypes[activeSegment].name} dialect... // Query optimization active // Protocol: {dbTypes[activeSegment].description} // Latency: 0.1ms // No code changes required // 
            </span>
          </div>
        </div>
      </div>

      <style dangerouslySetInnerHTML={{ __html: `
        @keyframes marquee {
          0% { transform: translateX(0%); }
          100% { transform: translateX(-100%); }
        }
        @keyframes moveAlongPath {
          from { offset-distance: 0%; }
          to { offset-distance: 100%; }
        }
        .path-animation {
          animation-name: moveAlongPath;
          animation-timing-function: linear;
          animation-iteration-count: infinite;
        }
      `}} />
    </div>
  );
};

export default App;