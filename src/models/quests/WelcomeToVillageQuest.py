from typing import List, Dict, Any
from src.models.Quest import Quest, QuestStep, QuestStepType, QuestReward, QuestType, QuestRegistry

class WelcomeToVillageQuest(Quest):
    id = "welcome_to_village"
    type = QuestType.MAIN

    def get_steps(self) -> List[QuestStep]:
        return [
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.welcome_to_village.step0_arrive",
                rewards=QuestReward(xp=50) 
            ),
            QuestStep(
                step_type=QuestStepType.DIALOGUE,
                text_key="quests.welcome_to_village.step1_elara",
                rewards=QuestReward(items=["health_potion"])
            ),
            QuestStep(
                step_type=QuestStepType.ACTIVITY,
                text_key="quests.welcome_to_village.step2_daily",
                rewards=QuestReward(money=100, achievement_id="quest_arrival"),
                target_stat="daily_uses",
                target_count=1
            )
        ]

    def on_activity(self, user_id: int, activity: str, amount: int, current_progress: Dict[str, Any]) -> bool:
        step_idx = current_progress.get('step_index', 0)
        if step_idx != 2: # The activity step is at index 2
            return False
            
        step = self.steps[step_idx]
        target_stat = step.extra.get('target_stat')
        if activity != target_stat:
            return False
            
        new_val = current_progress.get('progress_value', 0) + amount
        from src.database.quest import update_quest_progress
        update_quest_progress(user_id, self.id, progress_value=new_val)
        
        return new_val >= step.extra.get('target_count', 1)

    def get_unlocks(self) -> List[str]:
        return ["rusty_reels"]

# Register the quest
QuestRegistry.register(WelcomeToVillageQuest)
