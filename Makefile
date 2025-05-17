.PHONY: run-lottery run-host switch install-air grpcui

# 安裝 air 工具
install-air:
	@echo "安裝 air 工具..."
	@go install github.com/air-verse/air@latest

# 安裝 grpcui 工具
install-grpcui:
	@echo "安裝 grpcui 工具..."
	@go install github.com/fullstorydev/grpcui/cmd/grpcui@latest

# 啟動彩票服務
run-lottery:
	@./scripts/run_lottery.sh

# 啟動主持人服務
run-host:
	@./scripts/run_host.sh

# 切換服務
switch:
	@./scripts/switch_service.sh

# 啟動 gRPC UI
# 使用方式：
#   - 基本用法：make grpcui
#   - 指定主機和端口：make grpcui GRPC_HOST=127.0.0.1 GRPC_PORT=9100
#   - 指定 Web UI 端口：make grpcui GRPC_WEB_PORT=8080
#   - 不自動打開瀏覽器：make grpcui OPEN_BROWSER=false
grpcui:
	@./scripts/run_grpcui.sh 