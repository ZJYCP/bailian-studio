import { NavLink, Route, Routes } from 'react-router-dom'
import { ImageIcon, Sparkles, ListTodo, FolderOpen, Cpu, Settings } from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { AuthGate } from '@/components/AuthGate'
import Studio from '@/pages/Studio'
import Tasks from '@/pages/Tasks'
import Assets from '@/pages/Assets'
import Models from '@/pages/Models'
import SettingsPage from '@/pages/Settings'

function navClass({ isActive }: { isActive: boolean }) {
  return `flex items-center gap-2 rounded-md px-3 py-2 text-sm transition-colors ${
    isActive ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
  }`
}

export default function App() {
  return (
    <AuthGate>
      <Layout />
    </AuthGate>
  )
}

function Layout() {
  const { data: providers } = useQuery({ queryKey: ['providers'], queryFn: api.listProviders })
  const noProvider = providers && providers.length === 0

  return (
    <div className="flex min-h-screen">
      <aside className="flex w-52 shrink-0 flex-col border-r bg-card px-3 py-4">
        <div className="mb-6 flex items-center gap-2 px-2">
          <Sparkles className="size-5 text-primary" />
          <div>
            <div className="text-sm font-semibold">百炼创作平台</div>
            <div className="text-xs text-muted-foreground">Bailian Studio</div>
          </div>
        </div>
        <nav className="flex flex-col gap-1">
          <NavLink to="/" end className={navClass}>
            <ImageIcon className="size-4" /> 创作中心
          </NavLink>
          <NavLink to="/tasks" className={navClass}>
            <ListTodo className="size-4" /> 任务列表
          </NavLink>
          <NavLink to="/assets" className={navClass}>
            <FolderOpen className="size-4" /> 资产库
          </NavLink>
          <NavLink to="/models" className={navClass}>
            <Cpu className="size-4" /> 模型管理
          </NavLink>
          <NavLink to="/settings" className={navClass}>
            <Settings className="size-4" /> 设置
          </NavLink>
        </nav>
        <div className="mt-auto px-2 text-xs text-muted-foreground">
          {noProvider ? (
            <span className="text-destructive">请先在「设置」配置 API Key</span>
          ) : (
            providers?.find((p) => p.is_default) && (
              <span>
                当前服务：{providers.find((p) => p.is_default)!.name}
                <br />
                {providers.find((p) => p.is_default)!.key_mask}
              </span>
            )
          )}
        </div>
      </aside>
      <main className="min-w-0 flex-1 overflow-x-hidden">
        <Routes>
          <Route path="/" element={<Studio />} />
          <Route path="/tasks" element={<Tasks />} />
          <Route path="/assets" element={<Assets />} />
          <Route path="/models" element={<Models />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Routes>
      </main>
    </div>
  )
}
