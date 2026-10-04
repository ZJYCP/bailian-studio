import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2, Lock, LogOut, Sparkles } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api } from '@/lib/api'

// 访问门：服务端配置了 ACCESS_TOKEN 且未登录时，拦截整个应用
export function AuthGate({ children }: { children: React.ReactNode }) {
  const { data, isLoading } = useQuery({
    queryKey: ['auth-status'],
    queryFn: api.authStatus,
    retry: false,
  })

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center text-muted-foreground">
        <Loader2 className="mr-2 size-4 animate-spin" /> 加载中…
      </div>
    )
  }
  if (data?.auth_required && !data.authenticated) {
    return <LoginScreen />
  }
  return <>{children}</>
}

function LoginScreen() {
  const [token, setToken] = useState('')
  const login = useMutation({
    mutationFn: () => api.login(token),
    onSuccess: () => {
      window.location.reload()
    },
    onError: () => toast.error('口令错误'),
  })
  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/40 p-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <Sparkles className="mx-auto mb-2 size-8 text-primary" />
          <CardTitle>百炼创作平台</CardTitle>
          <CardDescription>请输入访问口令</CardDescription>
        </CardHeader>
        <CardContent>
          <form
            className="space-y-3"
            onSubmit={(e) => {
              e.preventDefault()
              if (token) login.mutate()
            }}
          >
            <div className="space-y-1.5">
              <Label htmlFor="token" className="sr-only">
                访问口令
              </Label>
              <Input
                id="token"
                type="password"
                autoFocus
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="访问口令"
              />
            </div>
            <Button type="submit" className="w-full" disabled={login.isPending || !token}>
              {login.isPending ? <Loader2 className="size-4 animate-spin" /> : <Lock className="size-4" />}
              解锁
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

// 设置页/侧栏用的锁定按钮
export function LogoutButton({ className }: { className?: string }) {
  const qc = useQueryClient()
  const logout = useMutation({
    mutationFn: api.logout,
    onSuccess: () => {
      qc.clear()
      window.location.reload()
    },
  })
  return (
    <Button variant="outline" size="sm" className={className} onClick={() => logout.mutate()}>
      <LogOut className="size-4" /> 锁定平台
    </Button>
  )
}
