from typing import List, Dict, Any
from src.models.Quest import Quest, QuestStep, QuestStepType, QuestReward, QuestType, QuestRegistry
from src.database.quest import update_quest_progress

class DailyQuest(Quest):
    id = "daily_quest"
    type = QuestType.DAILY

    OBJECTIVES = [
        {"stat": "blackjack_won", "count": 3, "text_key": "quests.daily.blackjack"},
        {"stat": "items_mined", "count": 10, "text_key": "quests.daily.mining"},
        {"stat": "items_fished", "count": 10, "text_key": "quests.daily.fishing"},
        {"stat": "slots_won", "count": 5, "text_key": "quests.daily.slots"},
        {"stat": "wagers_won", "count": 2, "text_key": "quests.daily.betting"},
    ]

    def get_steps(self) -> List[QuestStep]:
        # Reward is always a seasonal egg
        reward = QuestReward(items=["œuf saison"], xp=100)
        return [
            QuestStep(
                step_type=QuestStepType.ACTIVITY,
                text_key="quests.daily_challenge.objective",
                rewards=reward
            )
        ]

    def on_activity(self, user_id: int, activity: str, amount: int, current_progress: Dict[str, Any]) -> bool:
        step_idx = current_progress.get('step_index', 0)
        if step_idx != 0:
            return False
            
        custom_data = current_progress.get('custom_data', {})
        target_stat = custom_data.get('target_stat')
        target_count = custom_data.get('target_count', 1)
        
        if activity != target_stat:
            return False
            
        new_val = current_progress.get('progress_value', 0) + amount
        update_quest_progress(user_id, self.id, progress_value=new_val)
        
        return new_val >= target_count

    def get_title(self, lang: str) -> str:
        from src.utils.i18n import t
        return t("quests.daily_challenge.title", lang)

QuestRegistry.register(DailyQuest)
