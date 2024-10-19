# 啟動與結束 API Gateway

Bifrost 支持 command line 的啟動、檢查等功能，完整的 API Gateway

## 啟動

| command 參數  |     預設     |           說明           |
| :------------ | :----------: | :---------------------- |
| -d, --daemon  |    false     |    啟動模式為背景模式    |
| -t, --test    |    false     | 檢測配置文件格式是否正確 |
| -c, --conf    | empty string |    啟動模式為背景模式    |
| -u, --upgrade |    false     |     進行程序的熱更新     |

## 結束

這邊將介紹如何做到 API Gateway 升級，在 Bifrost 升級分為兩個部分

SIGINT: fast shutdown

當接收到 SIGINT (ctrl + c) 信號時，服務器將立即退出，沒有延遲，所有未完成的請求將被中斷。這種行為比較不推薦，因為它可能會中斷請求。

SIGTERM: graceful shutdown

當收到 SIGTERM 信號時，伺服器會通知所有服務關閉，等待預設的時間後退出，這種行為給予請求一個完成的寬限期之後優雅退出。

SIGQUIT: graceful upgrade

當伺服器接收到類似於 SIGTERM 的信號時，它會將所有正在監聽的 socket 轉移到新的 Bifrost instance，以確保升級過程中沒有停機時間，詳情請參閱優雅升級部分。
