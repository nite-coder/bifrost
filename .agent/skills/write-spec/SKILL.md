---
name: write-tech-spec
description: Write technical specification for a feature or module.
---


# Spec Template (規格書樣板)
當要求你撰寫規格書或技術方案時，請嚴格按照以下格式：
1. **背景 (Background)**: 
   - 說明需求的背景與核心痛點。
   - 領域名詞定義 (Domain Context)，確保術語一致性。
2. **目標 (Objectives)**: 
   - 說明要達到的願景，與這個願景如何解決背景中的問題與痛點
3. **現況分析 (Current State & Gap Analysis)**: 
   - 分析現有代碼或架構瓶頸，說明目前的情況與目標的差距。
4. **技術方案 (Technical Solution)**: 
   - **架構設計**: 組件關係與數據流向。
   - **核心實作**: 數據結構選擇與設計模式。
   - **性能考量**: 內存分配優化、序列化開銷、Context Switch 減少。
5. **驗證與測試策略 (Validation)**: 
   - 說明如何驗證功能正確性。
   - 包含 Benchmark 指標與數據一致性校驗（Balance Check）策略。
6. **注意事項 (Caveats & Trade-offs)**: 
   - 說明過程中發現的重要問題、風險反饋以及方案的限制。