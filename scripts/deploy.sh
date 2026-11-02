#!/bin/bash
# ============================================================
# go-star 生产部署脚本
#
# 前提：go-star-linux.new 已上传到本脚本所在目录
# 用法：./deploy.sh
#
# 流程：前置检查 → 备份 → 迁移 → 替换 → 停旧进程 → 启动 → 健康检查
# 任何一步失败都会中止，迁移失败时旧服务不受影响
# 健康检查失败会自动回滚到备份
# ============================================================

set -euo pipefail

APP_DIR="/home/admin/go-star"
BIN_NAME="go-star-linux"
NEW_BIN="go-star-linux.new"
CONF="settings.prod.yaml"
APP_LOG="app.log"
HEALTH_URL="http://127.0.0.1:8080/api/banners"

cd "$APP_DIR" || { echo "❌ 目录不存在: $APP_DIR"; exit 1; }

NOW=$(date '+%Y-%m-%d %H:%M:%S')
BAK_NAME="go-star-linux.bak.$(date +%Y%m%d-%H%M%S)"

echo "=========================================="
echo " go-star 部署开始 $NOW"
echo "=========================================="

# ---------- 0. 前置检查 ----------
echo ""
echo "【0/6】前置检查"

if [ ! -f "$NEW_BIN" ]; then
    echo "❌ 找不到 $NEW_BIN，请先上传"
    exit 1
fi

HEADER=$(head -c 4 "$NEW_BIN" | od -An -tx1 | tr -d ' \n')
if [ "$HEADER" != "7f454c46" ]; then
    echo "❌ $NEW_BIN 不是 Linux ELF 文件（文件头=$HEADER）"
    exit 1
fi
echo "✅ 新二进制是有效的 ELF 文件"

if [ ! -f "$CONF" ]; then
    echo "❌ 找不到配置文件 $CONF"
    exit 1
fi

chmod +x "$NEW_BIN"
NEW_MD5=$(md5sum "$NEW_BIN" | awk '{print $1}')
echo ""
echo "   新二进制 MD5 = $NEW_MD5"
echo "   ↑ 请与本地 Get-FileHash 的结果核对（大小写不敏感）"
echo ""
read -p "   MD5 一致吗？输入 yes 继续: " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    echo "❌ 已取消，未做任何改动"
    exit 1
fi

# ---------- 1. 备份 ----------
echo ""
echo "【1/6】备份旧二进制"

if [ ! -f "$BIN_NAME" ]; then
    echo "❌ 找不到 $BIN_NAME"
    exit 1
fi

cp -a "$BIN_NAME" "$BAK_NAME"
OLD_MD5=$(md5sum "$BIN_NAME" | awk '{print $1}')
BAK_MD5=$(md5sum "$BAK_NAME" | awk '{print $1}')
if [ "$OLD_MD5" != "$BAK_MD5" ]; then
    echo "❌ 备份校验失败：$OLD_MD5 != $BAK_MD5"
    exit 1
fi
echo "✅ 已备份 → $BAK_NAME"
echo "   MD5 = $OLD_MD5"

# ---------- 2. 查找旧进程 ----------
echo ""
echo "【2/6】查找运行中的进程"

OLD_PID=$(pgrep -f "$BIN_NAME -f $CONF" | head -1 || true)
if [ -z "$OLD_PID" ]; then
    echo "⚠️  没有找到运行中的进程，将直接启动"
else
    echo "✅ 旧进程 PID = $OLD_PID"
fi

# ---------- 3. 数据库迁移 ----------
echo ""
echo "【3/6】执行数据库迁移"
echo "   ⚠️  注意：-db 模式的退出码恒为 0，只能靠日志判断"

MIG_LOG=$(mktemp)
./"$NEW_BIN" -f "$CONF" -db 2>&1 | tee "$MIG_LOG" || true

if grep -q "数据库迁移成功" "$MIG_LOG"; then
    echo "✅ 迁移成功"
else
    echo "❌ 迁移失败，部署中止（旧服务未受影响，二进制未替换）"
    rm -f "$MIG_LOG"
    exit 1
fi
rm -f "$MIG_LOG"

# ---------- 4. 替换二进制 ----------
echo ""
echo "【4/6】替换二进制"
mv -f "$NEW_BIN" "$BIN_NAME"
echo "✅ 已替换"
ls -l "$BIN_NAME"

# ---------- 5. 停止旧进程 ----------
echo ""
echo "【5/6】停止旧进程"
if [ -n "$OLD_PID" ]; then
    kill -TERM "$OLD_PID"
    STOPPED=0
    for i in $(seq 1 30); do
        if ! kill -0 "$OLD_PID" 2>/dev/null; then
            echo "✅ 已优雅退出（等待 $((i * 5))00ms）"
            STOPPED=1
            break
        fi
        sleep 0.5
    done
    if [ "$STOPPED" != "1" ]; then
        echo "⚠️  15 秒后仍未退出，强制结束（浏览历史队列可能丢数据）"
        kill -9 "$OLD_PID" 2>/dev/null || true
        sleep 1
    fi
else
    echo "   无旧进程，跳过"
fi

# ---------- 6. 启动 + 健康检查 ----------
echo ""
echo "【6/6】启动新版本"

nohup ./"$BIN_NAME" -f "$CONF" > "$APP_LOG" 2>&1 &
NEW_PID=$!
echo "   新进程 PID = $NEW_PID"
echo "   等待服务就绪（最多 15 秒）..."

OK=0
for i in $(seq 1 30); do
    sleep 0.5
    CODE=$(curl -s -o /dev/null -w '%{http_code}' --max-time 2 "$HEALTH_URL" 2>/dev/null || echo "000")
    if [ "$CODE" = "200" ]; then
        echo "✅ 健康检查通过（HTTP 200，等待 $((i * 5))00ms）"
        OK=1
        break
    fi
done

if [ "$OK" != "1" ]; then
    echo ""
    echo "❌ 健康检查失败，最近 30 行日志："
    echo "------------------------------------------"
    tail -30 "$APP_LOG" || true
    echo "------------------------------------------"
    echo ""
    echo "开始自动回滚..."
    kill -TERM "$NEW_PID" 2>/dev/null || true
    sleep 2
    kill -9 "$NEW_PID" 2>/dev/null || true
    cp -a "$BAK_NAME" "$BIN_NAME"
    nohup ./"$BIN_NAME" -f "$CONF" > "$APP_LOG" 2>&1 &
    ROLLBACK_PID=$!
    echo "   回滚进程 PID = $ROLLBACK_PID"
    sleep 4
    RB_CODE=$(curl -s -o /dev/null -w '%{http_code}' --max-time 3 "$HEALTH_URL" 2>/dev/null || echo "000")
    if [ "$RB_CODE" = "200" ]; then
        echo "✅ 已回滚，旧版本恢复正常运行"
    else
        echo "🔴 回滚后仍不健康（HTTP=$RB_CODE），请立即人工介入！"
        tail -30 "$APP_LOG" || true
    fi
    exit 1
fi

# ---------- 收尾 ----------
echo ""
echo "=========================================="
echo " ✅ 部署完成 $(date '+%Y-%m-%d %H:%M:%S')"
echo "=========================================="
echo " 新版 PID : $NEW_PID"
echo " 旧版备份 : $BAK_NAME"
echo " 应用日志 : $APP_DIR/$APP_LOG"
echo ""
echo "--- 启动日志（最后 20 行）---"
tail -20 "$APP_LOG" || true
echo ""
echo "--- 内存占用 ---"
ps -o pid,rss,etime,args -p "$NEW_PID" | sed 's/ -f .*//' || true
echo ""
echo "--- 指标验证 ---"
curl -s -o /dev/null -w '  /metrics HTTP=%{http_code}\n' http://127.0.0.1:8080/metrics || true
echo ""
echo "如需回滚："
echo "  cp -a $BAK_NAME $BIN_NAME && kill -TERM $NEW_PID && nohup ./$BIN_NAME -f $CONF > $APP_LOG 2>&1 &"
