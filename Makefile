.PHONY: run-lottery run-host switch install-air

# 安裝 air 工具
install-air:
	@echo "安裝 air 工具..."
	@go install github.com/cosmtrek/air@latest

# 啟動彩票服務
run-lottery:
	@./scripts/run_lottery.sh

# 啟動主持人服務
run-host:
	@./scripts/run_host.sh

# 切換服務
switch:
	@./scripts/switch_service.sh 